package tournament

import (
	"context"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const singleElimination = "single_elimination"

type Bracket struct {
	ChampionID *uuid.UUID    `json:"championId"`
	Slots      []BracketSlot `json:"slots"`
}

type BracketSlot struct {
	ID           uuid.UUID  `json:"id"`
	Round        int        `json:"round"`
	Position     int        `json:"position"`
	Player1ID    *uuid.UUID `json:"player1Id"`
	Player1      *string    `json:"player1"`
	Player2ID    *uuid.UUID `json:"player2Id"`
	Player2      *string    `json:"player2"`
	WinnerID     *uuid.UUID `json:"winnerId"`
	FixtureID    *uuid.UUID `json:"fixtureId"`
	MatchID      *uuid.UUID `json:"matchId"`
	NextSlotID   *uuid.UUID `json:"nextSlotId"`
	NextSlotSide *int       `json:"nextSlotSide"`
	Status       string     `json:"status"`
}

func readBracket(ctx context.Context, tx pgx.Tx, summary Summary) (*Bracket, error) {
	rows, err := tx.Query(ctx, `select b.id, b.round_number as round, b.position,
		b.player1_id, p1.username as player1, b.player2_id, p2.username as player2,
		coalesce(b.winner_id,m.winner_id) as winner_id, b.fixture_id, f.match_id, next.id as next_slot_id,
		case when next.id is not null then 2-b.position%2 end as next_slot_side,
		case when b.winner_id is not null and b.fixture_id is null then 'bye'
			when coalesce(b.winner_id,m.winner_id) is not null then 'finished'
			when $2='completed' then 'closed'
			when m.status in ('finished','cancelled') then 'replay_required'
			when m.status is not null then m.status
			when b.fixture_id is not null then 'ready'
			else 'waiting' end as status
		from tournament_bracket_slots b
		left join users p1 on p1.id=b.player1_id
		left join users p2 on p2.id=b.player2_id
		left join tournament_fixtures f on f.id=b.fixture_id
		left join matches m on m.id=f.match_id
		left join tournament_bracket_slots next on next.tournament_id=b.tournament_id
			and next.round_number=b.round_number+1 and next.position=(b.position+1)/2
		where b.tournament_id=$1 order by b.round_number,b.position`, summary.ID, summary.Status)
	if err != nil {
		return nil, err
	}
	slots, err := pgx.CollectRows(rows, pgx.RowToStructByName[BracketSlot])
	if err != nil {
		return nil, err
	}
	bracket := &Bracket{Slots: slots}
	if len(slots) > 0 && summary.Status == "completed" {
		bracket.ChampionID = slots[len(slots)-1].WinnerID
	}
	return bracket, nil
}

func seedOrder(players int) []int {
	seeds := []int{1, 2}
	for len(seeds) < players {
		size := len(seeds) * 2
		next := make([]int, 0, size)
		for _, seed := range seeds {
			next = append(next, seed, size+1-seed)
		}
		seeds = next
	}
	return seeds
}

func seedPlayer(players []uuid.UUID, seed int) *uuid.UUID {
	if seed > len(players) {
		return nil
	}
	return &players[seed-1]
}

func createBracket(ctx context.Context, tx pgx.Tx, id uuid.UUID) (bool, error) {
	rows, err := tx.Query(ctx, `select user_id from tournament_participants
		where tournament_id=$1 order by random(),user_id`, id)
	if err != nil {
		return false, err
	}
	players, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil || len(players) < 2 {
		return false, err
	}
	seeds := seedOrder(len(players))
	slots := make([][]any, 0, len(seeds)-1)
	for round, positions := 1, len(seeds)/2; positions > 0; round, positions = round+1, positions/2 {
		for position := 1; position <= positions; position++ {
			var p1, p2, bye *uuid.UUID
			if round == 1 {
				p1 = seedPlayer(players, seeds[2*position-2])
				p2 = seedPlayer(players, seeds[2*position-1])
				if p1 == nil {
					bye = p2
				} else if p2 == nil {
					bye = p1
				}
			}
			slots = append(slots, []any{id, round, position, p1, p2, bye})
		}
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"tournament_bracket_slots"},
		[]string{"tournament_id", "round_number", "position", "player1_id", "player2_id", "winner_id"}, pgx.CopyFromRows(slots)); err != nil {
		return false, err
	}
	if err := advanceBracketSlots(ctx, tx, id); err != nil {
		return false, err
	}
	return true, createBracketFixtures(ctx, tx, id)
}

func advanceBracketSlots(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, `update tournament_bracket_slots parent
		set player1_id=first.winner_id, player2_id=second.winner_id
		from tournament_bracket_slots first
		join tournament_bracket_slots second on second.tournament_id=first.tournament_id
			and second.round_number=first.round_number and second.position=first.position+1
		where parent.tournament_id=$1 and first.tournament_id=parent.tournament_id
			and first.round_number=parent.round_number-1 and first.position=2*parent.position-1
			and (parent.player1_id,parent.player2_id) is distinct from (first.winner_id,second.winner_id)`, id)
	return err
}

func createBracketFixtures(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, `with ready as (
		select id, least(player1_id,player2_id) as player1_id, greatest(player1_id,player2_id) as player2_id
		from tournament_bracket_slots where tournament_id=$1 and fixture_id is null
			and player1_id is not null and player2_id is not null
	), created as (
		insert into tournament_fixtures (tournament_id,player1_id,player2_id)
		select $1,player1_id,player2_id from ready returning id,player1_id,player2_id
	)
	update tournament_bracket_slots b set fixture_id=created.id from ready
	join created using (player1_id,player2_id) where b.id=ready.id`, id)
	return err
}

func applyBracketWinners(ctx context.Context, tx pgx.Tx, id uuid.UUID) (bool, bool, error) {
	rows, err := tx.Query(ctx, `update tournament_bracket_slots b set winner_id=m.winner_id
		from tournament_fixtures f join matches m on m.id=f.match_id
		where b.tournament_id=$1 and b.fixture_id=f.id and b.winner_id is null
			and m.status='finished' and m.winner_id is not null returning b.round_number`, id)
	if err != nil {
		return false, false, err
	}
	decided, err := pgx.CollectRows(rows, pgx.RowTo[int])
	if err != nil || len(decided) == 0 {
		return false, false, err
	}
	if err := advanceBracketSlots(ctx, tx, id); err != nil {
		return false, false, err
	}
	var finalRound int
	if err := tx.QueryRow(ctx, "select max(round_number) from tournament_bracket_slots where tournament_id=$1",
		id).Scan(&finalRound); err != nil {
		return false, false, err
	}
	champion := slices.Contains(decided, finalRound)
	if champion {
		_, err = tx.Exec(ctx, "with "+bracketProgressSQL+`
			update tournament_participants p set bracket_rank=b.rank from bracket_ranks b
			where p.tournament_id=$1 and p.user_id=b.user_id`, id)
	}
	return true, champion, err
}

const bracketHasDecisiveMatch = `exists (
	select 1 from tournament_bracket_slots b join tournament_fixtures f on f.id=b.fixture_id
	join matches m on m.id=f.match_id where b.tournament_id=t.id
	and b.winner_id is null and m.status='finished' and m.winner_id is not null
)`

const bracketRanksSQL = ", " + bracketProgressSQL

const bracketProgressSQL = `bracket_progress as (
	select p.user_id,max(b.round_number) as reached,
		bool_or(b.winner_id is not distinct from p.user_id and next.id is null) as champion,
		bool_or(m.status='finished' and m.winner_id is not null) as has_decisive_match
	from tournament_participants p join tournament_bracket_slots b on b.tournament_id=p.tournament_id
		and p.user_id in (b.player1_id,b.player2_id)
	left join tournament_fixtures f on f.id=b.fixture_id
	left join matches m on m.id=f.match_id
	left join tournament_bracket_slots next on next.tournament_id=b.tournament_id
		and next.round_number=b.round_number+1 and next.position=(b.position+1)/2
	where p.tournament_id=$1 group by p.user_id
), bracket_ranks as (
	select user_id,dense_rank() over (order by champion desc,reached desc) as rank,
		has_decisive_match,bool_or(champion) over () as has_champion
	from bracket_progress
)`
