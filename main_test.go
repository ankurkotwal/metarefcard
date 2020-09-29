package main

import (
	"testing"

	"github.com/ankurkotwal/MetaRefCard/metarefcard"
)

func TestMain(m *testing.M) {
	metarefcard.RunLocal(files)
}

func TestFS2020(t *testing.T) {
	metarefcard.RunLocal(files)
}
func BenchmarkFS2020(b *testing.B) {
	b.Logf("BenchmarkFS2020")
	for n := 0; n < b.N; n++ {
		metarefcard.RunLocal(files)
	}
}

var files []string = []string{
	"testdata/x55_stick.xml",
	"testdata/x55_throttle.xml",
	"testdata/default_fs2020/Joystick_-_HOTAS_Warthog.xml",
	"testdata/default_fs2020/Logitech_Extreme_3D.xml",
	"testdata/default_fs2020/Saitek_Pro_Flight_Rudder_Pedals.xml",
	"testdata/default_fs2020/Saitek_Pro_Flight_X-56_Rhino_Stick.xml",
	"testdata/default_fs2020/Saitek_Pro_Flight_X-56_Rhino_Throttle.xml",
	"testdata/default_fs2020/Saitek_X52_Flight_Control_System.xml",
	"testdata/default_fs2020/Saitek_X52_Pro_Flight_Control_System.xml",
	"testdata/default_fs2020/Throttle_-_HOTAS_Warthog.xml",
	"testdata/default_fs2020/TWCS_Throttle.xml",
	"testdata/default_fs2020/T.16000M.xml",
	"testdata/default_fs2020/T.Flight_Hotas_4.xml",
	"testdata/default_fs2020/T.Flight_Hotas_One.xml",
	"testdata/default_fs2020/T.Flight_Stick_X.xml",
	"testdata/default_fs2020/T.Flight_Rudder_Pedals.xml",
	"testdata/default_fs2020/T.Flight_Hotas_X.xml",
}
