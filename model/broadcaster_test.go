package model

import (
	. "github.com/smartystreets/goconvey/convey"
	"sync"
	"testing"
)

type Listener struct {
	HasBeenCalled bool
}

func Test_Broadcaster(t *testing.T) {

	var b *Broadcaster

	Convey("Given a broadcaster", t, func() {
		var wg sync.WaitGroup

		b = NewBroadcaster()

		l1 := &Listener{HasBeenCalled: false}
		l2 := &Listener{HasBeenCalled: false}
		l3 := &Listener{HasBeenCalled: false}

		channel1 := b.Listen()
		channel2 := b.Listen()
		wg.Add(2)

		go func() {
			defer wg.Done()
			<-channel1
			l1.HasBeenCalled = true
		}()

		go func() {
			defer wg.Done()
			<-channel2
			l2.HasBeenCalled = true
		}()

		So(l1.HasBeenCalled, ShouldEqual, false)
		So(l2.HasBeenCalled, ShouldEqual, false)
		So(l3.HasBeenCalled, ShouldEqual, false)

		Convey("When one send an event to it", func() {
			b.Write(&struct{}{})
			wg.Wait()

			Convey("Then the listener has received the message", func() {
				So(l1.HasBeenCalled, ShouldEqual, true)
				So(l2.HasBeenCalled, ShouldEqual, true)
				So(l3.HasBeenCalled, ShouldEqual, false)

			})

		})
	})

}
