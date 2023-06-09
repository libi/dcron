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

func (s *IRecentJobPackerTestSuite) TestNormal() {
	timeLimit := time.Second * 6
	wg := &sync.WaitGroup{}
	wg.Add(1)
	rencentPacker := dcron.NewRecentJobPacker(timeLimit)
	go func() {
		<-time.After(8 * time.Second)
		jobNames := rencentPacker.PopAllJobs()
		expectJobNames := make([]string, 0)
		for i := 4; i < 16; i++ {
			expectJobNames = append(expectJobNames, "i_"+strconv.Itoa(i))
		}
		s.Equal(expectJobNames, jobNames)
		wg.Done()
	}()
	for i := 0; i < 16; i++ {
		rencentPacker.AddJob("i_"+strconv.Itoa(i), time.Now())
		<-time.After(500 * time.Millisecond)
	}
	wg.Wait()
}

func TestIRecentJobPackerTestSuite(t *testing.T) {
	s := new(IRecentJobPackerTestSuite)
	suite.Run(t, s)
}
