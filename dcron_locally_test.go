package dcron_test

import (
	"testing"
	"time"

	"github.com/libi/dcron"
	"github.com/libi/dcron/cron"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DcronLocallyTestSuite struct {
	suite.Suite
}

func (s *DcronLocallyTestSuite) TestNormal() {
	dcr := dcron.NewDcronWithOption(
		"not a necessary servername",
		nil,
		dcron.RunningLocally(),
		dcron.CronOptionSeconds(),
		dcron.CronOptionChain(
			cron.Recover(
				cron.DefaultLogger,
			)))

	s.Assert().NotNil(dcr)
	runningTime := 60 * time.Second
	t := s.T()
	var err error
	err = dcr.AddFunc("job1", "*/5 * * * * *", func() {
		t.Log(time.Now())
	})
	require.Nil(t, err)
	err = dcr.AddFunc("job2", "*/8 * * * * *", func() {
		panic("test panic")
	})
	require.Nil(t, err)
	err = dcr.AddFunc("job3", "*/2 * * * * *", func() {
		t.Log("job3:", time.Now())
	})
	require.Nil(t, err)
	dcr.Start()
	<-time.After(runningTime)
	dcr.Stop()
}

func TestDcronLocallyTestSuite(t *testing.T) {
	suite.Run(t, &DcronLocallyTestSuite{})
}
