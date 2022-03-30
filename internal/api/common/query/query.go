package query

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"

	"rightsizing-api-server/internal/utils"
)

// Query 파라미터들 parsing 하기 위해 사용함
type parseQuery struct {
	Namespace string `query:"namespace,omitempty" description:"the namespace of pod"`
	StartTime string `query:"start,omitempty" json:"-"`
	EndTime   string `query:"end,omitempty" json:"-"`
	Forecast  string `query:"forecast,omitempty" json:"-"`
}

type Query struct {
	ID        string
	Namespace string
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Forecast  bool
}

func (q parseQuery) ParseAndValidate(c *fiber.Ctx) (Query, error) {
	var (
		id        = c.Locals("requestid").(string)
		namespace = c.Params("namespace", "")
		name      = c.Params("name", "")
		// default time
		startTime = time.Now().AddDate(0, 0, -7)
		endTime   = time.Now()
	)

	if q.StartTime != "" {
		start, err := utils.TimeParser(q.StartTime)
		if err != nil {
			return Query{}, err
		}
		startTime = start
	}

	if q.EndTime != "" {
		end, err := utils.TimeParser(q.EndTime)
		if err != nil {
			return Query{}, err
		}
		endTime = end
	}

	if startTime.After(endTime) {
		return Query{}, errors.New("the end time should be after the start time")
	}

	return Query{
		ID:        id,
		Namespace: namespace,
		Name:      name,
		StartTime: startTime,
		EndTime:   endTime,
	}, nil
}

func ParseAndValidate(c *fiber.Ctx) (Query, error) {
	query := &parseQuery{}
	if err := c.QueryParser(query); err != nil {
		return Query{}, err
	}
	return query.ParseAndValidate(c)
}
