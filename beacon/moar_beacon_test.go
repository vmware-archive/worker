package beacon_test

import (
	"os"
	. "github.com/concourse/worker/beacon"
	"github.com/concourse/worker/beacon/beaconfakes"
	"code.cloudfoundry.org/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/concourse/baggageclaim/baggageclaimfakes"
	"github.com/concourse/atc"
	"errors"
)

var _ = FDescribe("Beacon", func() {

	var (
		beacon        Beacon
		fakeClient    *beaconfakes.FakeClient
		fakeSession   *beaconfakes.FakeSession
		fakeCloseable *beaconfakes.FakeCloseable
		fakeVolumeOne *baggageclaimfakes.FakeVolume
		fakeVolumeTwo *baggageclaimfakes.FakeVolume
	)

	BeforeEach(func() {
		fakeClient = new(beaconfakes.FakeClient)
		fakeSession = new(beaconfakes.FakeSession)
		fakeCloseable = new(beaconfakes.FakeCloseable)
		fakeVolumeOne = new(baggageclaimfakes.FakeVolume)
		fakeVolumeTwo = new(baggageclaimfakes.FakeVolume)
		fakeClient.NewSessionReturns(fakeSession, nil)
		fakeClient.DialReturns(fakeCloseable, nil)
		logger := lager.NewLogger("test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))

		beacon = Beacon{
			KeepAlive: true,
			Logger:    logger,
			Client:    fakeClient,
			Worker: atc.Worker{
				GardenAddr:      "1.2.3.4:7777",
				BaggageclaimURL: "wat://5.6.7.8:7788",
			},
		}
	})
	var _ = Describe("Land", func() {

		var (
			signals chan os.Signal
			ready   chan<- struct{}
		)

		BeforeEach(func() {
			signals = make(chan os.Signal)
			ready = make(chan struct{})
		})

		AfterEach(func() {
			Expect(fakeCloseable.CloseCallCount()).To(Equal(1))
		})

		Context("when waiting on the remote command takes some time", func() {
			var (
				keepAliveErr    chan error
				cancelKeepAlive chan struct{}
				wait            chan bool
				landErr         chan error
			)

			JustBeforeEach(func() {
				go func() {
					landErr <- beacon.LandWorker(signals, make(chan struct{}, 1))
					close(landErr)
				}()
			})

			BeforeEach(func() {
				keepAliveErr = make(chan error, 1)
				cancelKeepAlive = make(chan struct{}, 1)
				wait = make(chan bool, 1)
				landErr = make(chan error)

				fakeSession.WaitStub = func() error {
					<-wait
					return nil
				}

				fakeClient.KeepAliveReturns(keepAliveErr, cancelKeepAlive)
			})

			It("closes the session and waits for it to shut down", func() {
				Consistently(landErr).ShouldNot(BeClosed()) // should be blocking on exit channel

				go func() {
					wait <- false
				}()

				Eventually(landErr).Should(Receive()) // should stop blocking
				Expect(fakeSession.CloseCallCount()).To(Equal(1))
			})

			Context("when keeping the connection alive errors", func() {
				var (
					err = errors.New("keepalive fail")
				)

				BeforeEach(func() {
					fakeClient.KeepAliveReturns(keepAliveErr, cancelKeepAlive)
					go func() {
						keepAliveErr <- err
					}()
				})

				It("returns the error", func() {
					Eventually(landErr).Should(Receive(&err))
				})
			})

		})

	})
})