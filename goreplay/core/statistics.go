package core

import (
	"record-traffic-press/goreplay/glogs"
	"record-traffic-press/goreplay/settings"

	"runtime"
	"strconv"
	"time"
)

type GorStat struct {
	statName string
	rateMs   int
	latest   int
	mean     int
	max      int
	count    int
}

func NewGorStat(statName string, rateMs int) (s *GorStat) {
	s = new(GorStat)
	s.statName = statName
	s.rateMs = rateMs
	s.latest = 0
	s.mean = 0
	s.max = 0
	s.count = 0

	if settings.Settings.Stats {
		go s.ReportStats()
	}
	return
}

func (s *GorStat) Write(latest int) {
	if settings.Settings.Stats {
		if latest > s.max {
			s.max = latest
		}
		if latest != 0 {
			s.mean = ((s.mean * s.count) + latest) / (s.count + 1)
		}
		s.latest = latest
		s.count = s.count + 1
	}
}

func (s *GorStat) Reset() {
	s.latest = 0
	s.max = 0
	s.mean = 0
	s.count = 0
}

func (s *GorStat) String() string {
	return s.statName + ":" + strconv.Itoa(s.latest) + "," + strconv.Itoa(s.mean) + "," + strconv.Itoa(s.max) + "," + strconv.Itoa(s.count) + "," + strconv.Itoa(s.count/(s.rateMs/1000.0)) + "," + strconv.Itoa(runtime.NumGoroutine())
}

func (s *GorStat) ReportStats() {
	glogs.Debug(0, "\n", s.statName+":latest,mean,max,count,count/second,gcount")
	for {
		glogs.Debug(0, "\n", s)
		s.Reset()
		time.Sleep(time.Duration(s.rateMs) * time.Millisecond)
	}
}
