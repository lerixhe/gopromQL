package tsdb

import (
	"fmt"
	"regexp"
	"strings"
)

type RangeFunctionType string
type AggOperatorType string

const (
	OneMinute  = "1m"
	FiveMinute = "5m"
	OneHour    = "1h"
	FiveHour   = "5h"
	OneDay     = "1d"
	OneWeek    = "7d"
	OneMonth   = "1M"
)

const (
	RangeFunctionIncrease RangeFunctionType = "increase"
	RangeFunctionRate     RangeFunctionType = "rate"
)
const (
	AggOperatorSum     AggOperatorType = "sum"
	AggOperatorMax     AggOperatorType = "max"
	AggOperatorMin     AggOperatorType = "min"
	AggOperatorAvg     AggOperatorType = "avg"
	AggOperatorTopk    AggOperatorType = "topk"
	AggOperatorBottomk AggOperatorType = "bottomk"
)

const (
	fmtLabel           = "{%s}"
	fmtRange           = "[%s]"
	fmtSubRange        = "%s(%s)[%s:%s]"
	FmtOffset          = " offset %s"
	fmtFunc            = "%s(%s)"
	fmtOpt             = "%s by(%s)(%s) "
	fmtLableMust       = `%s="%s",`
	fmtLableNot        = `%s!="%s",`
	fmtLableMustRegx   = `%s=~"%s",`
	fmtLableMustRegxIn = `%s=~"^(?:%s)$",` //正则中对IN查询有特殊优化，此处进行支持
	fmtLableNotRegx    = `%s!~"%s",`
)

type PromQLRaw struct {
	Name             string            //指标名称
	LablesMust       map[string]string //指标lables,必须匹配的
	LablesNot        map[string]string //指标lables,必须不匹配的
	LablesMustRegx   map[string]string //指标lables,必须符合指定正则的
	LablesMustRegxIn map[string]string //指标lables,必须符合In条件的
	LablesNotRegx    map[string]string //指标lables,必须不符合指定正则的
	RangeFunc        RangeFunctionType //range方法
	RangeTime        string            //range时间范围
	OffsetTime       string            //时间偏移量
	AggOperator      AggOperatorType   //聚合方法
	AggByLables      []string          //按指定labels聚合
	SubRange         *struct {         //不为空时，将原表达式拼装为子查询
		RangeFunc  RangeFunctionType //子range方法
		RangeTime  string            //子range时间范围
		Resolution string            //子range的分辨率（粒度）
	}
	OrPromQLRaw *PromQLRaw //多个查询语句用OR连接时使用
}

func (m *PromQLRaw) makePromQL() string {
	label := ""
	if len(m.LablesMust) != 0 {
		for k, v := range m.LablesMust {
			if len(v) == 0 {
				continue
			}
			label += fmt.Sprintf(fmtLableMust, k, v)
		}
	}
	if len(m.LablesNot) != 0 {
		for k, v := range m.LablesNot {
			if len(v) == 0 {
				continue
			}
			label += fmt.Sprintf(fmtLableNot, k, v)
		}
	}
	if len(m.LablesMustRegx) != 0 {
		for k, v := range m.LablesMustRegx {
			if len(v) == 0 {
				continue
			}
			label += fmt.Sprintf(fmtLableMustRegx, k, v)
		}
	}
	if len(m.LablesMustRegxIn) != 0 {
		for k, v := range m.LablesMustRegxIn {
			if len(v) == 0 {
				continue
			}
			label += fmt.Sprintf(fmtLableMustRegxIn, k, v)
		}
	}
	if len(m.LablesNotRegx) != 0 {
		for k, v := range m.LablesNotRegx {
			if len(v) == 0 {
				continue
			}
			label += fmt.Sprintf(fmtLableNotRegx, k, v)
		}
	}
	promQL := m.Name + fmt.Sprintf(fmtLabel, label)
	if m.RangeTime != "" {
		promQL += fmt.Sprintf(fmtRange, m.RangeTime)
	}
	if m.OffsetTime != "" {
		promQL += fmt.Sprintf(FmtOffset, m.OffsetTime)
	}
	if m.RangeFunc != "" {
		promQL = fmt.Sprintf(fmtFunc, m.RangeFunc, promQL)
	}
	if m.SubRange != nil {
		promQL = fmt.Sprintf(fmtSubRange, m.SubRange.RangeFunc, promQL, m.SubRange.RangeTime, m.SubRange.Resolution)
	}
	if m.AggOperator != "" {
		lableStr := ""
		for _, name := range m.AggByLables {
			lableStr += name + ","
		}
		promQL = fmt.Sprintf(fmtOpt, m.AggOperator, lableStr, promQL)
	}
	if m.OrPromQLRaw != nil {
		orPromQL := m.OrPromQLRaw.makePromQL()
		promQL += " or " + orPromQL
	}
	return promQL

}
func QuoteMeta(v string) string {
	v = regexp.QuoteMeta(v)              //go自带的处理转义的方式
	v = strings.ReplaceAll(v, `\`, `\\`) //在promQL中\本身也要转义，故需要多这一步
	return v
}

type Universal struct {
	ResultsType string             `json:"results_type"`
	ResultsLen  int                `json:"results_len"`
	Results     []*UniversalValues `json:"results"`
}

func NewUniversal() *Universal {
	return &Universal{
		Results: make([]*UniversalValues, 0),
	}
}

type UniversalValues struct {
	Values    map[int64]float64 `json:"values"`
	ValuesLen int               `json:"values_len"`
	Lables    map[string]string `json:"lables"`
}

func NewUniversalValues() *UniversalValues {
	return &UniversalValues{
		Values: make(map[int64]float64),
		Lables: make(map[string]string),
	}
}
