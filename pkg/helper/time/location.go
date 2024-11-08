package time

import pkgTime "time"

type LocationFunc func() *pkgTime.Location

var Location LocationFunc = locationFuncImpl

func locationFuncImpl() *pkgTime.Location {
	l, _ := pkgTime.LoadLocation("Asia/Jakarta")
	return l
}
