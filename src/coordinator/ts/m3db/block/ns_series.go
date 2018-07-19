// Copyright (c) 2018 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package block

import (
	"math"
	"time"

	"github.com/m3db/m3db/src/coordinator/block"
	"github.com/m3db/m3db/src/dbnode/encoding"
	"github.com/m3db/m3db/src/dbnode/ts"
	"github.com/m3db/m3x/ident"
)

// NSBlock is a single block for a given timeseries and namespace
// which contains all of the necessary SeriesIterators so that consolidation can
// happen across namespaces
type NSBlock struct {
	ID              ident.ID
	Namespace       ident.ID
	Bounds          block.Bounds
	SeriesIterators encoding.SeriesIterators
}

type nsBlockStepIter struct {
	m3dbIters        []encoding.SeriesIterator
	bounds           block.Bounds
	seriesIndex, idx int
	lastDP           ts.Datapoint
}

// Next moves to the next item
func (c *nsBlockStepIter) Next() bool {
	c.idx++
	// NB(braskin): this is inclusive of the last step in the iterator
	indexTime, err := c.bounds.TimeForIndex(c.idx)
	if err != nil { // index is out of bounds
		return false
	}

	lastDP := c.lastDP
	// NB(braskin): check to make sure that the current index time is after the last
	// seen datapoint and Next() on the underlaying m3db iterator returns true
	for indexTime.After(lastDP.Timestamp) && c.nextIterator() {
		lastDP, _, _ = c.m3dbIters[c.seriesIndex].Current()
		c.lastDP = lastDP
	}

	return true
}

func (c *nsBlockStepIter) nextIterator() bool {
	// todo(braskin): check bounds as well
	if len(c.m3dbIters) == 0 {
		return false
	}

	for c.seriesIndex < len(c.m3dbIters) {
		if c.m3dbIters[c.seriesIndex].Next() {
			return true
		}
		c.seriesIndex++
	}

	return false
}

// Current returns the float64 value for the current step
func (c *nsBlockStepIter) Current() float64 {
	lastDP := c.lastDP

	indexTime, err := c.bounds.TimeForIndex(c.idx)
	if err != nil {
		return math.NaN()
	}

	// NB(braskin): if the last datapoint is after the current step, but before the (current step+1),
	// return that datapoint, otherwise return NaN
	if !indexTime.After(lastDP.Timestamp) && indexTime.Add(c.bounds.StepSize).After(lastDP.Timestamp) {
		return lastDP.Value
	}

	return math.NaN()
}

// Close closes the underlaying iterators
func (c *nsBlockStepIter) Close() {
	// todo(braskin): implement this function
}

type nsBlockSeriesIter struct {
	m3dbIters        []encoding.SeriesIterator
	bounds           block.Bounds
	seriesIndex, idx int
	lastDP           ts.Datapoint
}

// Next moves to the next item and returns when it is out of bounds
func (c *nsBlockSeriesIter) Next() bool {
	_, err := c.bounds.TimeForIndex(c.idx)
	return err == nil
}

// Current returns the slice of values for the current series
func (c *nsBlockSeriesIter) Current() []float64 {
	var (
		vals        []float64
		indexTime   time.Time
		err         error
		outOfBounds bool
	)

	for _, iter := range c.m3dbIters {
		for iter.Next() {
			indexTime, err = c.bounds.TimeForIndex(c.idx)
			if err != nil {
				break
			}

			curr, _, _ := iter.Current()
			c.lastDP = curr

			// Keep adding NaNs while we reach the next datapoint
			if indexTime.After(c.lastDP.Timestamp) {
				continue
			}
			for indexTime.Before(c.lastDP.Timestamp) && !indexTime.Add(c.bounds.StepSize).After(c.lastDP.Timestamp) {
				indexTime, err = c.bounds.TimeForIndex(c.idx)
				if err != nil {
					outOfBounds = true
					break
				}
				if indexTime.Add(c.bounds.StepSize).After(c.lastDP.Timestamp) {
					break
				}
				vals = append(vals, math.NaN())
				c.idx++
			}
			if outOfBounds {
				break
			}
			vals = append(vals, c.lastDP.Value)
			c.idx++
		}
	}

	// once we have gone through all of the iterators, check to see if we need to add more NaNs
	for indexTime, err = c.bounds.TimeForIndex(c.idx); err == nil; indexTime, err = c.bounds.TimeForIndex(c.idx) {
		if indexTime.Equal(c.bounds.End.Add(c.bounds.StepSize)) {
			break
		}
		vals = append(vals, math.NaN())
		c.idx++
	}
	return vals
}

// Close closes the underlaying iterators
func (c *nsBlockSeriesIter) Close() {
	// todo(braskin): implement this function
}
