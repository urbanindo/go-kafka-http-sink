package time

import timePkg "time"

type NowFunc func() timePkg.Time

func Now() timePkg.Time {
	return timePkg.Now().In(Location())
}
