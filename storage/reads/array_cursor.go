package reads

import (
	"context"
	"fmt"

	"github.com/influxdata/flux/interval"
	"github.com/influxdata/influxdb/storage/reads/datatypes"
	"github.com/influxdata/influxdb/tsdb/cursors"
)

type singleValue struct {
	v interface{}
}

func (v *singleValue) Value(key string) (interface{}, bool) {
	return v.v, true
}

func newAggregateArrayCursor(ctx context.Context, agg []*datatypes.Aggregate, cursor cursors.Cursor) (cursors.Cursor, error) {
	switch agg[0].Type {
	case datatypes.Aggregate_AggregateTypeFirst, datatypes.Aggregate_AggregateTypeLast:
		return newLimitArrayCursor(cursor), nil
	}
	return NewWindowAggregateArrayCursor(ctx, agg, interval.Window{}, cursor)
}

func NewWindowAggregateArrayCursor(ctx context.Context, agg []*datatypes.Aggregate, window interval.Window, cursor cursors.Cursor) (cursors.Cursor, error) {
	if cursor == nil {
		return nil, nil
	}

	switch agg[0].Type {
	case datatypes.Aggregate_AggregateTypeCount:
		return newWindowCountArrayCursor(cursor, window), nil
	case datatypes.Aggregate_AggregateTypeSum:
		return newWindowSumArrayCursor(cursor, window)
	case datatypes.Aggregate_AggregateTypeFirst:
		return newWindowFirstArrayCursor(cursor, window), nil
	case datatypes.Aggregate_AggregateTypeLast:
		return newWindowLastArrayCursor(cursor, window), nil
	case datatypes.Aggregate_AggregateTypeMin:
		return newWindowMinArrayCursor(cursor, window), nil
	case datatypes.Aggregate_AggregateTypeMax:
		return newWindowMaxArrayCursor(cursor, window), nil
	case datatypes.Aggregate_AggregateTypeMean:
		if len(agg) == 2 && agg[1].Type == datatypes.Aggregate_AggregateTypeCount {
			return newWindowMeanCountArrayCursor(cursor, window)
		}
		return newWindowMeanArrayCursor(cursor, window)

	default:
		// TODO(sgc): should be validated higher up
		panic("invalid aggregate")
	}
}

type cursorContext struct {
	ctx  context.Context
	req  *cursors.CursorRequest
	itrs cursors.CursorIterators
	err  error
}

type multiShardArrayCursors struct {
	ctx context.Context
	req cursors.CursorRequest

	cursors struct {
		i integerMultiShardArrayCursor
		f floatMultiShardArrayCursor
		u unsignedMultiShardArrayCursor
		b booleanMultiShardArrayCursor
		s stringMultiShardArrayCursor
	}
}

// newMultiShardArrayCursors is a factory for creating cursors for each series key.
// The range of the cursor is [start, end). The start time is the lower absolute time
// and the end time is the higher absolute time regardless of ascending or descending order.
func newMultiShardArrayCursors(ctx context.Context, start, end int64, asc bool) *multiShardArrayCursors {
	// When we construct the CursorRequest, we translate the time range
	// from [start, stop) to [start, stop]. The cursor readers from storage are
	// inclusive on both ends and we perform that conversion here.
	m := &multiShardArrayCursors{
		ctx: ctx,
		req: cursors.CursorRequest{
			Ascending: asc,
			StartTime: start,
			EndTime:   end - 1,
		},
	}

	cc := cursorContext{
		ctx: ctx,
		req: &m.req,
	}

	m.cursors.i.cursorContext = cc
	m.cursors.f.cursorContext = cc
	m.cursors.u.cursorContext = cc
	m.cursors.b.cursorContext = cc
	m.cursors.s.cursorContext = cc

	return m
}

func (m *multiShardArrayCursors) createCursor(row SeriesRow) cursors.Cursor {
	m.req.Name = row.Name
	m.req.Tags = row.SeriesTags
	m.req.Field = row.Field

	var cond expression
	if row.ValueCond != nil {
		cond = &astExpr{row.ValueCond}
	}

	var shard cursors.CursorIterator
	var cur cursors.Cursor
	var err error
	for cur == nil && len(row.Query) > 0 {
		shard, row.Query = row.Query[0], row.Query[1:]
		cur, err = shard.Next(m.ctx, &m.req)
	}

	if cur == nil || err != nil {
		return nil
	}

	switch c := cur.(type) {
	case cursors.IntegerArrayCursor:
		m.cursors.i.reset(c, row.Query, cond)
		return &m.cursors.i
	case cursors.FloatArrayCursor:
		m.cursors.f.reset(c, row.Query, cond)
		return &m.cursors.f
	case cursors.UnsignedArrayCursor:
		m.cursors.u.reset(c, row.Query, cond)
		return &m.cursors.u
	case cursors.StringArrayCursor:
		m.cursors.s.reset(c, row.Query, cond)
		return &m.cursors.s
	case cursors.BooleanArrayCursor:
		m.cursors.b.reset(c, row.Query, cond)
		return &m.cursors.b
	default:
		panic(fmt.Sprintf("unreachable: %T", cur))
	}
}

// createCursorForSelectors creates a cursor that gathers ALL shard cursors for selector
// aggregates (first, last, min, max). This ensures correct results when a series spans
// multiple shards or TSI partitions.
func (m *multiShardArrayCursors) createCursorForSelectors(row SeriesRow, agg []*datatypes.Aggregate) (cursors.Cursor, error) {
	m.req.Name = row.Name
	m.req.Tags = row.SeriesTags
	m.req.Field = row.Field

	// Gather ALL cursors from ALL shards
	var (
		intCursors    []cursors.IntegerArrayCursor
		floatCursors  []cursors.FloatArrayCursor
		uintCursors   []cursors.UnsignedArrayCursor
		stringCursors []cursors.StringArrayCursor
		boolCursors  []cursors.BooleanArrayCursor
	)

	for _, shard := range row.Query {
		cur, err := shard.Next(m.ctx, &m.req)
		if err != nil || cur == nil {
			continue
		}
		switch c := cur.(type) {
		case cursors.IntegerArrayCursor:
			intCursors = append(intCursors, c)
		case cursors.FloatArrayCursor:
			floatCursors = append(floatCursors, c)
		case cursors.UnsignedArrayCursor:
			uintCursors = append(uintCursors, c)
		case cursors.StringArrayCursor:
			stringCursors = append(stringCursors, c)
		case cursors.BooleanArrayCursor:
			boolCursors = append(boolCursors, c)
		}
	}

	// Determine which cursor type we have and create appropriate merged cursor
	if len(intCursors) > 0 {
		return newMergedIntegerLimitArrayCursor(intCursors, agg[0].Type), nil
	}
	if len(floatCursors) > 0 {
		return newMergedFloatLimitArrayCursor(floatCursors, agg[0].Type), nil
	}
	if len(uintCursors) > 0 {
		return newMergedUnsignedLimitArrayCursor(uintCursors, agg[0].Type), nil
	}
	if len(stringCursors) > 0 {
		return newMergedStringLimitArrayCursor(stringCursors, agg[0].Type), nil
	}
	if len(boolCursors) > 0 {
		return newMergedBooleanLimitArrayCursor(boolCursors, agg[0].Type), nil
	}

	return nil, nil
}

// mergedLimitArrayCursor is a cursor that merges multiple shard cursors
// for selector aggregates. It collects the first value from each cursor
// and returns the one with the max timestamp (for last) or min timestamp (for first).
type mergedLimitArrayCursor interface {
	cursors.Cursor
}

func newMergedFloatLimitArrayCursor(curs []cursors.FloatArrayCursor, aggType datatypes.Aggregate_AggregateType) *mergedFloatLimitArrayCursor {
	return &mergedFloatLimitArrayCursor{
		cursors: curs,
		aggType: aggType,
		res:     cursors.NewFloatArrayLen(1),
	}
}

type mergedFloatLimitArrayCursor struct {
	cursors []cursors.FloatArrayCursor
	aggType datatypes.Aggregate_AggregateType
	res     *cursors.FloatArray
}

func (c *mergedFloatLimitArrayCursor) Stats() cursors.CursorStats {
	var stats cursors.CursorStats
	for _, cur := range c.cursors {
		stats.Add(cur.Stats())
	}
	return stats
}

func (c *mergedFloatLimitArrayCursor) Next() *cursors.FloatArray {
	// Collect first value from each cursor
	type cursorValue struct {
		cursorIdx int
		timestamp int64
		value     float64
	}
	var values []cursorValue
	for i, cur := range c.cursors {
		a := cur.Next()
		if len(a.Timestamps) > 0 {
			values = append(values, cursorValue{
				cursorIdx: i,
				timestamp: a.Timestamps[0],
				value:     a.Values[0],
			})
		}
	}

	if len(values) == 0 {
		return &cursors.FloatArray{}
	}

	// Find the value with max/min timestamp
	var selected cursorValue
	if c.aggType == datatypes.Aggregate_AggregateTypeLast {
		// For last, we want max timestamp
		selected = values[0]
		for _, v := range values[1:] {
			if v.timestamp > selected.timestamp {
				selected = v
			}
		}
	} else {
		// For first, we want min timestamp
		selected = values[0]
		for _, v := range values[1:] {
			if v.timestamp < selected.timestamp {
				selected = v
			}
		}
	}

	c.res.Timestamps = []int64{selected.timestamp}
	c.res.Values = []float64{selected.value}
	return c.res
}

func (c *mergedFloatLimitArrayCursor) Close() {
	for _, cur := range c.cursors {
		cur.Close()
	}
}

func (c *mergedFloatLimitArrayCursor) Err() error {
	return nil
}

func newMergedIntegerLimitArrayCursor(curs []cursors.IntegerArrayCursor, aggType datatypes.Aggregate_AggregateType) *mergedIntegerLimitArrayCursor {
	return &mergedIntegerLimitArrayCursor{
		cursors: curs,
		aggType: aggType,
		res:     cursors.NewIntegerArrayLen(1),
	}
}

type mergedIntegerLimitArrayCursor struct {
	cursors []cursors.IntegerArrayCursor
	aggType datatypes.Aggregate_AggregateType
	res     *cursors.IntegerArray
}

func (c *mergedIntegerLimitArrayCursor) Stats() cursors.CursorStats {
	var stats cursors.CursorStats
	for _, cur := range c.cursors {
		stats.Add(cur.Stats())
	}
	return stats
}

func (c *mergedIntegerLimitArrayCursor) Next() *cursors.IntegerArray {
	type cursorValue struct {
		cursorIdx int
		timestamp int64
		value     int64
	}
	var values []cursorValue
	for i, cur := range c.cursors {
		a := cur.Next()
		if len(a.Timestamps) > 0 {
			values = append(values, cursorValue{
				cursorIdx: i,
				timestamp: a.Timestamps[0],
				value:     a.Values[0],
			})
		}
	}

	if len(values) == 0 {
		return &cursors.IntegerArray{}
	}

	var selected cursorValue
	if c.aggType == datatypes.Aggregate_AggregateTypeLast {
		selected = values[0]
		for _, v := range values[1:] {
			if v.timestamp > selected.timestamp {
				selected = v
			}
		}
	} else {
		selected = values[0]
		for _, v := range values[1:] {
			if v.timestamp < selected.timestamp {
				selected = v
			}
		}
	}

	c.res.Timestamps = []int64{selected.timestamp}
	c.res.Values = []int64{selected.value}
	return c.res
}

func (c *mergedIntegerLimitArrayCursor) Close() {
	for _, cur := range c.cursors {
		cur.Close()
	}
}

func (c *mergedIntegerLimitArrayCursor) Err() error {
	return nil
}

func newMergedUnsignedLimitArrayCursor(curs []cursors.UnsignedArrayCursor, aggType datatypes.Aggregate_AggregateType) *mergedUnsignedLimitArrayCursor {
	return &mergedUnsignedLimitArrayCursor{
		cursors: curs,
		aggType: aggType,
		res:     cursors.NewUnsignedArrayLen(1),
	}
}

type mergedUnsignedLimitArrayCursor struct {
	cursors []cursors.UnsignedArrayCursor
	aggType datatypes.Aggregate_AggregateType
	res     *cursors.UnsignedArray
}

func (c *mergedUnsignedLimitArrayCursor) Stats() cursors.CursorStats {
	var stats cursors.CursorStats
	for _, cur := range c.cursors {
		stats.Add(cur.Stats())
	}
	return stats
}

func (c *mergedUnsignedLimitArrayCursor) Next() *cursors.UnsignedArray {
	type cursorValue struct {
		cursorIdx int
		timestamp int64
		value     uint64
	}
	var values []cursorValue
	for i, cur := range c.cursors {
		a := cur.Next()
		if len(a.Timestamps) > 0 {
			values = append(values, cursorValue{
				cursorIdx: i,
				timestamp: a.Timestamps[0],
				value:     a.Values[0],
			})
		}
	}

	if len(values) == 0 {
		return &cursors.UnsignedArray{}
	}

	var selected cursorValue
	if c.aggType == datatypes.Aggregate_AggregateTypeLast {
		selected = values[0]
		for _, v := range values[1:] {
			if v.timestamp > selected.timestamp {
				selected = v
			}
		}
	} else {
		selected = values[0]
		for _, v := range values[1:] {
			if v.timestamp < selected.timestamp {
				selected = v
			}
		}
	}

	c.res.Timestamps = []int64{selected.timestamp}
	c.res.Values = []uint64{selected.value}
	return c.res
}

func (c *mergedUnsignedLimitArrayCursor) Close() {
	for _, cur := range c.cursors {
		cur.Close()
	}
}

func (c *mergedUnsignedLimitArrayCursor) Err() error {
	return nil
}

func newMergedStringLimitArrayCursor(curs []cursors.StringArrayCursor, aggType datatypes.Aggregate_AggregateType) *mergedStringLimitArrayCursor {
	return &mergedStringLimitArrayCursor{
		cursors: curs,
		aggType: aggType,
		res:     cursors.NewStringArrayLen(1),
	}
}

type mergedStringLimitArrayCursor struct {
	cursors []cursors.StringArrayCursor
	aggType datatypes.Aggregate_AggregateType
	res     *cursors.StringArray
}

func (c *mergedStringLimitArrayCursor) Stats() cursors.CursorStats {
	var stats cursors.CursorStats
	for _, cur := range c.cursors {
		stats.Add(cur.Stats())
	}
	return stats
}

func (c *mergedStringLimitArrayCursor) Next() *cursors.StringArray {
	type cursorValue struct {
		cursorIdx int
		timestamp int64
		value     string
	}
	var values []cursorValue
	for i, cur := range c.cursors {
		a := cur.Next()
		if len(a.Timestamps) > 0 {
			values = append(values, cursorValue{
				cursorIdx: i,
				timestamp: a.Timestamps[0],
				value:     a.Values[0],
			})
		}
	}

	if len(values) == 0 {
		return &cursors.StringArray{}
	}

	var selected cursorValue
	if c.aggType == datatypes.Aggregate_AggregateTypeLast {
		selected = values[0]
		for _, v := range values[1:] {
			if v.timestamp > selected.timestamp {
				selected = v
			}
		}
	} else {
		selected = values[0]
		for _, v := range values[1:] {
			if v.timestamp < selected.timestamp {
				selected = v
			}
		}
	}

	c.res.Timestamps = []int64{selected.timestamp}
	c.res.Values = []string{selected.value}
	return c.res
}

func (c *mergedStringLimitArrayCursor) Close() {
	for _, cur := range c.cursors {
		cur.Close()
	}
}

func (c *mergedStringLimitArrayCursor) Err() error {
	return nil
}

func newMergedBooleanLimitArrayCursor(curs []cursors.BooleanArrayCursor, aggType datatypes.Aggregate_AggregateType) *mergedBooleanLimitArrayCursor {
	return &mergedBooleanLimitArrayCursor{
		cursors: curs,
		aggType: aggType,
		res:     cursors.NewBooleanArrayLen(1),
	}
}

type mergedBooleanLimitArrayCursor struct {
	cursors []cursors.BooleanArrayCursor
	aggType datatypes.Aggregate_AggregateType
	res     *cursors.BooleanArray
}

func (c *mergedBooleanLimitArrayCursor) Stats() cursors.CursorStats {
	var stats cursors.CursorStats
	for _, cur := range c.cursors {
		stats.Add(cur.Stats())
	}
	return stats
}

func (c *mergedBooleanLimitArrayCursor) Next() *cursors.BooleanArray {
	type cursorValue struct {
		cursorIdx int
		timestamp int64
		value     bool
	}
	var values []cursorValue
	for i, cur := range c.cursors {
		a := cur.Next()
		if len(a.Timestamps) > 0 {
			values = append(values, cursorValue{
				cursorIdx: i,
				timestamp: a.Timestamps[0],
				value:     a.Values[0],
			})
		}
	}

	if len(values) == 0 {
		return &cursors.BooleanArray{}
	}

	var selected cursorValue
	if c.aggType == datatypes.Aggregate_AggregateTypeLast {
		selected = values[0]
		for _, v := range values[1:] {
			if v.timestamp > selected.timestamp {
				selected = v
			}
		}
	} else {
		selected = values[0]
		for _, v := range values[1:] {
			if v.timestamp < selected.timestamp {
				selected = v
			}
		}
	}

	c.res.Timestamps = []int64{selected.timestamp}
	c.res.Values = []bool{selected.value}
	return c.res
}

func (c *mergedBooleanLimitArrayCursor) Close() {
	for _, cur := range c.cursors {
		cur.Close()
	}
}

func (c *mergedBooleanLimitArrayCursor) Err() error {
	return nil
}
