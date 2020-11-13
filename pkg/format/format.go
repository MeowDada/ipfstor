package format

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Col denotes a column of data.
type Col struct {
	Key   string
	Value interface{}
}

// Row consists of an array of columns.
type Row []Col

// Keys return all lables of this row.
func (r Row) Keys() (strs []string) {
	for i := range r {
		strs = append(strs, r[i].Key)
	}
	return strs
}

// Values returns an array of values.
func (r Row) Values() (vs []interface{}) {
	for i := range r {
		vs = append(vs, r[i].Value)
	}
	return vs
}

// ValueStrings returns an arry of value strings.
func (r Row) ValueStrings() (strs []string) {
	for i := range r {
		strs = append(strs, fmt.Sprintf("%v", r[i].Value))
	}
	return strs
}

// Template is an instance to print out columns.
type Template interface {
	Render(cols []Row) []byte
}

// Options denotes a render options.
type Options struct {
	Sort bool
}

// Basic is the simplest render template.
type Basic struct {
	basic
}

type basic struct{}

func (basic) Render(rows []Row, opts Options) []byte {
	if len(rows) == 0 {
		return nil
	}

	// Sort each rows in ascending order.
	if opts.Sort {
		sort.Slice(rows, func(i, j int) bool {
			if len(rows[i]) == 0 {
				return false
			}
			return rows[i][0].Key < rows[j][0].Key
		})
	}

	// Create labels.
	labels := rows[0].Keys()

	numLabels := len(labels)
	if numLabels == 0 {
		return nil
	}

	// Find longest string len in each columns.
	lenLabels := make([]int, numLabels)
	for i := range labels {
		lenLabels[i] = len(labels[i])
	}

	for _, row := range rows {
		vals := row.ValueStrings()
		for i := range vals {
			if len(vals[i]) > lenLabels[i] {
				lenLabels[i] = len(vals[i])
			}
		}
	}

	// Construct label format string
	labelFormat := make([]string, numLabels)
	valueFormat := make([]string, numLabels)

	for i := range labels {
		labelFormat[i] = "%-" + strconv.Itoa(lenLabels[i]) + "s"
		valueFormat[i] = "%-" + strconv.Itoa(lenLabels[i]) + "v"
	}

	labelFormatStr := "|" + strings.Join(labelFormat, "|") + "|"
	valueFormatStr := "|" + strings.Join(valueFormat, "|") + "|"

	labelVals := make([]interface{}, numLabels)
	for i := 0; i < numLabels; i++ {
		labelVals[i] = labels[i]
	}

	labelStr := fmt.Sprintf(labelFormatStr, labelVals...)

	emptyVals := make([]interface{}, numLabels)
	for i := range emptyVals {
		emptyVals[i] = strings.Repeat("-", lenLabels[i])
	}

	banner := fmt.Sprintf(labelFormatStr, emptyVals...)

	ret := banner + "\n" + labelStr + "\n" + banner + "\n"

	for i := range rows {
		ret += fmt.Sprintf(valueFormatStr, rows[i].Values()...) + "\n"
	}

	ret += banner + "\n"

	return bytes.NewBufferString(ret).Bytes()
}
