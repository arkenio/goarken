package drivers

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func Test_RancherDriver(t *testing.T) {

	Convey("Given a rancher host", t, func() {
		rancherHost := "http://192.168.99.106:8080/v1/projects/1a5"

		projectId := getProjectIdFromRancherHost(rancherHost)
		Convey("Then i can extract its project id", func() {
			So(projectId, ShouldEqual, "1a5")
		})
	})
}
