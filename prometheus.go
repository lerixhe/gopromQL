
package tsdb

/*
使用了prometheus官方库，兼容性更好，底层为HTTP协议
*/
import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
	
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
)

var promClient api.Client
var promOnce sync.Once

func GetPromClient() api.Client {
	var err error
	promOnce.Do(func() {
		conf := config.Global()
		promClient, err = api.NewClient(api.Config{
			Address: "http://" + conf.Datasources.Tsdb.Addr + conf.Datasources.Tsdb.HTTPPort,
		})
		if err != nil {
			log.Errorf("Error creating client: %v", err)
			os.Exit(1)
		}
	})
	return promClient

}

// QueryFromPrometheus 对时序数据从当前时刻offset前的时刻开始，对最近的指标数据进行查询
func (p *PromQLRaw) QueryFromPrometheus(qtime time.Time) (*Universal, error) {
	promClient = GetPromClient()
	v1api := v1.NewAPI(promClient)
	ctx, cancel := context.WithTimeout(context.Background(), config.Global().Datasources.Tsdb.Timeout.Duration)
	defer cancel()
	promQL := p.makePromQL()
	log.Debugf("promQL:%s", promQL)
	result, warnings, err := v1api.Query(ctx, promQL, qtime)
	if err != nil {
		return nil, fmt.Errorf("error QueryFromTSDB: %v,promQL:%s", err, promQL)
	}
	if len(warnings) > 0 {
		log.Warnf("Warnings: %v", warnings)
	}
	return ParseModelValues(result), nil
}

// QueryRangeFromPrometheus 对时序数据从当前时刻offset前的时刻开始，每隔step从进行X分钟的范围查询，进而求出此范围内的增量、速率均值等
func (p *PromQLRaw) QueryRangeFromPrometheus(start, end time.Time, step time.Duration) (*Universal, error) {
	promClient = GetPromClient()
	v1api := v1.NewAPI(promClient)
	ctx, cancel := context.WithTimeout(context.Background(), config.Global().Datasources.Tsdb.Timeout.Duration)
	defer cancel()
	r := v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	}
	promQL := p.makePromQL()
	log.Debugf("promQL:%s", promQL)
	result, warnings, err := v1api.QueryRange(ctx, promQL, r)
	if len(warnings) > 0 {
		log.Warnf("Warnings: %v", warnings)
	}
	if err != nil {
		log.Errorf("err:%s", err)
		return nil, fmt.Errorf("error QueryRangeFromTSDB: %v,promQL:%s", err, promQL)
	}

	// log.Debugf("%+v", result)
	log.Debugf("%+v", result.Type().String())
	return ParseModelValues(result), nil
}

func ParseModelValues(v model.Value) *Universal {
	universal := NewUniversal()
	switch result := v.(type) {
	case model.Matrix:
		universal.ResultsType = "matrix"
		for _, v := range result {
			uv := NewUniversalValues()
			for _, vv := range v.Values {
				uv.Values[vv.Timestamp.Unix()] = float64(vv.Value)
			}
			uv.ValuesLen = len(uv.Values)
			if d := v.Metric; len(d) != 0 {
				for n, v := range d {
					uv.Lables[string(n)] = string(v)
				}
			}
			universal.Results = append(universal.Results, uv)
		}
		universal.ResultsLen = len(universal.Results)
	case model.Vector:
		universal.ResultsType = "vector"
		for _, v := range result {
			uv := NewUniversalValues()
			uv.Values[v.Timestamp.Unix()] = float64(v.Value)
			uv.ValuesLen = len(uv.Values)
			if d := v.Metric; len(d) != 0 {
				for n, v := range d {
					uv.Lables[string(n)] = string(v)
				}
			}
			universal.Results = append(universal.Results, uv)
		}
		universal.ResultsLen = len(universal.Results)
	default:
		universal.ResultsType = "others"
		universal.Results = append(universal.Results, NewUniversalValues())
	}
	return universal
}
