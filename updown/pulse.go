package updown

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2025 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"time"

	"github.com/essentialkaos/ek/v13/req"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Webhook is updown pulse URL
var Webhook string

// ////////////////////////////////////////////////////////////////////////////////// //

// Pulse sends request to Updown.io pulse
func Pulse(payload string) error {
	if Webhook == "" {
		return nil
	}

	rt := req.NewRetrier()
	r := req.Request{URL: Webhook, Method: req.GET, AutoDiscard: true}

	if payload != "" {
		r.Method = req.POST
		r.Body = payload
	}

	_, err := rt.Do(r, req.Retry{
		Num:    5,
		Pause:  time.Second / 2,
		Status: req.STATUS_OK,
	})

	if err != nil {
		return fmt.Errorf("Can't send request to updown pulse URL: %v", err)
	}

	return nil
}
