package dcron_test

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/libi/dcron"
	"github.com/stretchr/testify/suite"
)

type IRecentJobPackerTestSuite struct {
	suite.Suite
}

func (s *IRecentJobPackerTestSuite) checkPopJobs(l, r int, wg *sync.WaitGroup, tAfter time.Duration, recentPacker dcron.IRecentJobPacker) {
	<-time.After(tAfter)
	jobNames := recentPacker.PopAllJobs()
	expectJobNames := make([]string, 0)
	for i := l; i < r; i++ {
		expectJobNames = append(expectJobNames, "i_"+strconv.Itoa(i))
	}
	s.Equal(expectJobNames, jobNames)
	wg.Done()
}

func (s *IRecentJobPackerTestSuite) TestNormal() {
	// the timeWindow is 6s and we will add a job each 0.5s
	// and check the jobs list after 8s.
	// expect job name is i_4 ~ i_16
	timeLimit := time.Second * 6
	wg := &sync.WaitGroup{}
	wg.Add(1)
	recentPacker := dcron.NewRecentJobPacker(timeLimit)

	go s.checkPopJobs(4, 16, wg, 8*time.Second, recentPacker)

	for i := 0; i < 16; i++ {
		recentPacker.AddJob("i_"+strconv.Itoa(i), time.Now())
		<-time.After(500 * time.Millisecond)
	}
	wg.Wait()
}

func (s *IRecentJobPackerTestSuite) Test_CallMultiTimes() {
	timeLimit := time.Second * 6
	wg := &sync.WaitGroup{}
	wg.Add(2)
	recentPacker := dcron.NewRecentJobPacker(timeLimit)

	go s.checkPopJobs(4, 16, wg, 8*time.Second, recentPacker)
	go s.checkPopJobs(20, 32, wg, 16*time.Second, recentPacker)

	for i := 0; i < 32; i++ {
		recentPacker.AddJob("i_"+strconv.Itoa(i), time.Now())
		<-time.After(500 * time.Millisecond)
	}
	wg.Wait()
}

func TestIRecentJobPackerTestSuite(t *testing.T) {
	s := new(IRecentJobPackerTestSuite)
	suite.Run(t, s)
}
