package x

import "cmp"

func MinMax[T cmp.Ordered](v, vMin, vMax, vDef T) (r T) {
	if vMin == vMax && vMin != r {
		return vMin
	}
	if vMin > vMax {
		vMax, vMin = vMin, vMax
	}
	return min(max(cmp.Or(v, vDef), vMin), vMax)
}
