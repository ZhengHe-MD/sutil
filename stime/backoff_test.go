package stime

import (
	"testing"
	"time"
	"log"
)

func TestBackoffReset(t *testing.T) {
	log.Println("BackOffRest Begin")
	bo := NewBackOffCtrl(time.Second * 1, time.Second*10)

	go func() {
		bg := time.Now().Unix()
		bo.BackOff() // 0s
		log.Printf("BackOffRest routine %d", time.Now().Unix() - bg)
		bo.BackOff() // 1s
		log.Printf("BackOffRest routine %d", time.Now().Unix() - bg)
		bo.BackOff() // 2s
		log.Printf("BackOffRest routine %d", time.Now().Unix() - bg)
		bo.BackOff() // 1s
		log.Printf("BackOffRest routine %d", time.Now().Unix() - bg)
		if time.Now().Unix() - bg != 4 {
			t.Errorf("BackOffRest reset err")
		} else {
			log.Println("BackOffRest OK Reset")
		}
	}()

	time.Sleep(time.Second * time.Duration(4))
	bo.Reset()

	bg := time.Now().Unix()
	bo.BackOff() // 02
	bo.BackOff() // 1s

	if time.Now().Unix() - bg != 1 {
		t.Errorf("BackOffRest reset Continue err")
	} else {
		log.Println("BackOffRest OK Reset Continue")
	}


}


func TestBackoff(t *testing.T) {
	bo := NewBackOffCtrl(time.Second * 1, time.Second*10)

	for i := uint32(0); i < 8; i++ {
		log.Printf("BackOff %d Begin", i)
		bg := time.Now().Unix()

		if i == 0 {
			bo.BackOff()
		}
		bo.BackOff()
		intv := time.Now().Unix() - bg

		if intv > 10 {
			t.Errorf("BackOff ceil err %d", intv)
		}

		if intv != 1 << i {
			if 1 << i > 10 && intv == 10 {
				log.Printf("BackOff ceil %d", i)
			} else {
				t.Errorf("BackOff time %d err", i)
			}
		}

		log.Printf("BackOff %d End", i)

	}


}
