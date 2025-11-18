package db

import (
	"context"
	"database/sql"

	"github.com/lenchik-en/lbs_server/internal/api"
)

type Location struct {
	Point    Point
	Accuracy int
}

type Point struct {
	Lat float64
	Lon float64
}

func NewLocation(lat, lac float64, acc int) *Location {
	return &Location{
		Point: Point{
			Lat: lat,
			Lon: lac,
		},
		Accuracy: acc,
	}
}

func FindCell(ctx context.Context, db *sql.DB, c *api.Cell) (*Location, error) {

	query := `SELECT lat, lon FROM cells
		WHERE tech = $1
		  AND mcc = $2
		  AND mnc = $3
	`

	args := []interface{}{c.Tech}
	switch c.Tech {
	case :

	}


	//
	//
	//if lac != nil {
	//	query += " AND lac = $" + strconv.Itoa(argIndex)
	//	args = append(args, *lac)
	//	argIndex++
	//}
	//if tac != nil {
	//	query += " AND tac = $" + strconv.Itoa(argIndex)
	//	args = append(args, *tac)
	//	argIndex++
	//}
	//if cid != nil {
	//	query += " AND cid = $" + strconv.Itoa(argIndex)
	//	args = append(args, *cid)
	//	argIndex++
	//}

	query += " LIMIT 1"

	row := db.QueryRowContext(ctx, query, args...)

	var l Location
	err := row.Scan(&l.Point.Lat, &l.Point.Lon)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &l, nil
}
