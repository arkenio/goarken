package goarken

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func Test_status(t *testing.T) {
	var status *Status

	Convey("Given a status", t, func() {

		status = &Status{}

		Convey("When started equals expected equals current", func() {
			status.Expected = "started"
			status.Current = "started"

			Convey("The computed status should be started if alive", func() {
				status.Alive = "1"
				So(status.Compute(), ShouldEqual, STARTED_STATUS)
			})

			Convey("The computed status should be error if not alive", func() {
				status.Alive = ""
				So(status.Compute(), ShouldEqual, ERROR_STATUS)
			})

		})

		Convey("When started is expected and current is starting", func() {
			status.Expected = "started"
			status.Current = "starting"

			Convey("Then computed status should be starting", func() {
				So(status.Compute(), ShouldEqual, STARTING_STATUS)

			})

		})

		Convey("When stopped is expected and current is stopped", func() {
			status.Expected = "stopped"
			status.Current = "stopped"
			Convey("Then computed status should be stopped", func() {

				So(status.Compute(), ShouldEqual, STOPPED_STATUS)

			})

		})

		Convey("When stopped is expected and current is stopping", func() {
			status.Expected = "stopped"
			status.Current = "stopping"

			Convey("Then computed status should be starting", func() {

				So(status.Compute(), ShouldEqual, STOPPED_STATUS)

			})

		})

		Convey("When current is passivated", func() {
			status.Expected = "passivated"
			status.Current = "stopped"

			Convey("Then computed status should be passivated", func() {

				So(status.Compute(), ShouldEqual, PASSIVATED_STATUS)

			})

		})

		Convey("When status is nil", func() {
			var status *Status
			status = nil

			Convey("Then the computed status should be NA", func() {
				So(status.Compute(), ShouldEqual, NA_STATUS)
			})

		})

	})

}
