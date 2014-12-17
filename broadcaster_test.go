package goarken

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

type Listener struct {
	HasBeenCalled bool
}

func Test_Broadcaster(t *testing.T) {

	var b *Broadcaster

	Convey("Given a broadcaster", t, func() {
		b = NewBroadcaster()
		l1 := &Listener{HasBeenCalled: false}
		l2 := &Listener{HasBeenCalled: false}
		l3 := &Listener{HasBeenCalled: false}

		channel1 := b.Listen()
		channel2 := b.Listen()

		go func() {
			<-channel1
			l1.HasBeenCalled = true
		}()

		go func() {
			<-channel2
			l2.HasBeenCalled = true
		}()

		So(l1.HasBeenCalled, ShouldEqual, false)
		So(l2.HasBeenCalled, ShouldEqual, false)
		So(l3.HasBeenCalled, ShouldEqual, false)

		Convey("When one send an event to it", func() {
			b.Write(&struct{}{})
			time.Sleep(time.Second)

			Convey("Then the listener has received the message", func() {
				So(l1.HasBeenCalled, ShouldEqual, true)
				So(l2.HasBeenCalled, ShouldEqual, true)
				So(l3.HasBeenCalled, ShouldEqual, false)
			})

		})
	})

}
