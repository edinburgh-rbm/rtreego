// Copyright 2012 Daniel Connelly.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rtreego

import (
	"fmt"
	"math"
	"strings"
)

// DistError is an improper distance measurement.  It implements the error
// and is generated when a distance-related assertion fails.
type DistError float64

func (err DistError) Error() string {
	return "rtreego: improper distance"
}

// Point represents a point in 3-dimensional Euclidean space.
type Point [Dim]float64

// Dist computes the Euclidean distance between two points p and q.
func (p Point) Dist(q Point) float64 {
	dp := p.Sub(q)
	return math.Sqrt(dp.Dot(dp))
}

func (p Point) Dot(q Point) float64 {
	sum := 0.0
	for i := range p {
		sum += p[i] * q[i]
	}
	return sum
}

// sum computes p + q
func (p Point) Add(q Point) Point {
	var sum Point
	for i := range p {
		sum[i] = p[i] + q[i]
	}
	return sum
}

// sub computes p - q
func (p Point) Sub(q Point) Point {
	var diff Point
	for i := range p {
		diff[i] = p[i] - q[i]
	}
	return diff
}

// scale computes a * p
func (p Point) Scale(a float64) Point {
	var s Point
	for i := range p {
		s[i] = a * p[i]
	}
	return s
}

func (p Point) Norm() float64 {
	return math.Sqrt(p.Dot(p))
}

func (p Point) Unit() Point {
	return p.Scale(1/p.Norm())
}

// projection of p in the q direction
func (p Point) Proj(q Point) Point {
	return q.Unit().Scale(p.Dot(q))
}

// minDist computes the square of the distance from a point to a rectangle.
// If the point is contained in the rectangle then the distance is zero.
//
// Implemented per Definition 2 of "Nearest Neighbor Queries" by
// N. Roussopoulos, S. Kelley and F. Vincent, ACM SIGMOD, pages 71-79, 1995.
func (p Point) minDist(r *Rect) float64 {
	sum := 0.0
	for i, pi := range p {
		if pi < r.P[i] {
			d := pi - r.P[i]
			sum += d * d
		} else if pi > r.Q[i] {
			d := pi - r.Q[i]
			sum += d * d
		} else {
			sum += 0
		}
	}
	return sum
}

// minMaxDist computes the minimum of the maximum distances from p to points
// on r.  If r is the bounding box of some geometric objects, then there is
// at least one object contained in r within minMaxDist(p, r) of p.
//
// Implemented per Definition 4 of "Nearest Neighbor Queries" by
// N. Roussopoulos, S. Kelley and F. Vincent, ACM SIGMOD, pages 71-79, 1995.
func (p Point) minMaxDist(r *Rect) float64 {
	// by definition, MinMaxDist(p, r) =
	// min{1<=k<=n}(|pk - rmk|^2 + sum{1<=i<=n, i != k}(|pi - rMi|^2))
	// where rmk and rMk are defined as follows:

	rm := func(k int) float64 {
		if p[k] <= (r.P[k]+r.Q[k])/2 {
			return r.P[k]
		}
		return r.Q[k]
	}

	rM := func(k int) float64 {
		if p[k] >= (r.P[k]+r.Q[k])/2 {
			return r.P[k]
		}
		return r.Q[k]
	}

	// This formula can be computed in linear time by precomputing
	// S = sum{1<=i<=n}(|pi - rMi|^2).

	S := 0.0
	for i := range p {
		d := p[i] - rM(i)
		S += d * d
	}

	// Compute MinMaxDist using the precomputed S.
	min := math.MaxFloat64
	for k := range p {
		d1 := p[k] - rM(k)
		d2 := p[k] - rm(k)
		d := S - d1*d1 + d2*d2
		if d < min {
			min = d
		}
	}

	return min
}

func (p Point) String() string {
	var s [Dim]string
	for i := 0; i < Dim; i++ {
		s[i] = fmt.Sprintf("%f", p[i])
	}
	return fmt.Sprintf("(%s)", strings.Join(s[:], ", "))
}

// Rect represents a subset of 3-dimensional Euclidean space of the form
// [a1, b1] x [a2, b2] x ... x [an, bn], where ai < bi for all 1 <= i <= n.
type Rect struct {
	P, Q Point // Enforced by NewRect: p[i] <= q[i] for all i.
}

func (r *Rect) String() string {
	var s [Dim]string
	for i, a := range r.P {
		b := r.Q[i]
		s[i] = fmt.Sprintf("[%.2f, %.2f]", a, b)
	}
	return strings.Join(s[:], "x")
}

// NewRect constructs and returns a pointer to a Rect given a corner point and
// the lengths of each dimension.  The point p should be the most-negative point
// on the rectangle (in every dimension) and every length should be positive.
func NewRect(p Point, lengths [Dim]float64) (r Rect, err error) {
	r.P = p
	r.Q = lengths
	for i, l := range r.Q {
		if l <= 0 {
			return r, DistError(l)
		}
		r.Q[i] += r.P[i]
	}
	return r, nil
}

// size computes the measure of a rectangle (the product of its side lengths).
func (r *Rect) size() float64 {
	size := 1.0
	for i, a := range r.P {
		b := r.Q[i]
		size *= b - a
	}
	return size
}

// margin computes the sum of the edge lengths of a rectangle.
func (r *Rect) margin() float64 {
	// The number of edges in an n-dimensional rectangle is n * 2^(n-1)
	// (http://en.wikipedia.org/wiki/Hypercube_graph).  Thus the number
	// of edges of length (ai - bi), where the rectangle is determined
	// by p = (a1, a2, ..., an) and q = (b1, b2, ..., bn), is 2^(n-1).
	//
	// The margin of the rectangle, then, is given by the formula
	// 2^(n-1) * [(b1 - a1) + (b2 - a2) + ... + (bn - an)].
	sum := 0.0
	for i, a := range r.P {
		b := r.Q[i]
		sum += b - a
	}
	return 4.0 * sum
}

// ContainsPoint tests whether p is located inside or on the boundary of r.
func (r *Rect) ContainsPoint(p Point) bool {
	for i, a := range p {
		// p is contained in (or on) r if and only if p <= a <= q for
		// every dimension.
		if a < r.P[i] || a > r.Q[i] {
			return false
		}
	}

	return true
}

// containsRect tests whether r2 is is located inside r1.
func (r1 *Rect) ContainsRect(r2 *Rect) bool {
	for i, a1 := range r1.P {
		b1, a2, b2 := r1.Q[i], r2.P[i], r2.Q[i]
		// enforced by constructor: a1 <= b1 and a2 <= b2.
		// so containment holds if and only if a1 <= a2 <= b2 <= b1
		// for every dimension.
		if a1 > a2 || b2 > b1 {
			return false
		}
	}

	return true
}

func (r1 *Rect) enlarge(r2 *Rect) {
	for i := 0; i < Dim; i++ {
		if r1.P[i] > r2.P[i] {
			r1.P[i] = r2.P[i]
		}
		if r1.Q[i] < r2.Q[i] {
			r1.Q[i] = r2.Q[i]
		}
	}
}

// intersect computes the intersection of two rectangles.  If no intersection
// exists, the intersection is nil.
func Intersect(r1, r2 *Rect) bool {
	// There are four cases of overlap:
	//
	//     1.  a1------------b1
	//              a2------------b2
	//              p--------q
	//
	//     2.       a1------------b1
	//         a2------------b2
	//              p--------q
	//
	//     3.  a1-----------------b1
	//              a2-------b2
	//              p--------q
	//
	//     4.       a1-------b1
	//         a2-----------------b2
	//              p--------q
	//
	// Thus there are only two cases of non-overlap:
	//
	//     1. a1------b1
	//                    a2------b2
	//
	//     2.             a1------b1
	//        a2------b2
	//
	// Enforced by constructor: a1 <= b1 and a2 <= b2.  So we can just
	// check the endpoints.

	for i := 0; i < Dim; i++ {
		if r2.Q[i] <= r1.P[i] || r1.Q[i] <= r2.P[i] {
			return false
		}
	}
	return true
}

// ToRect constructs a rectangle containing p with side lengths 2*tol.
func (p Point) ToRect(tol float64) *Rect {
	var r Rect
	for i := range p {
		r.P[i] = p[i] - tol
		r.Q[i] = p[i] + tol
	}
	return &r
}

func initBoundingBox(r, r1, r2 *Rect) {
	*r = *r1
	r.enlarge(r2)
}

// boundingBox constructs the smallest rectangle containing both r1 and r2.
func boundingBox(r1, r2 *Rect) *Rect {
	var r Rect
	initBoundingBox(&r, r1, r2)
	return &r
}
