package maps

import (
	"container/heap"

	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func WalkPath(gat *res.GAT, fromX, fromY, toX, toY int) []worldstate.WalkStep {
	if fromX == toX && fromY == toY {
		return []worldstate.WalkStep{{X: fromX, Y: fromY}}
	}
	if gat == nil {
		return []worldstate.WalkStep{{X: fromX, Y: fromY}, {X: toX, Y: toY}}
	}
	if path, ok := FindWalkPath(gat, fromX, fromY, toX, toY); ok {
		return path
	}
	return nil
}

// Classic servers allow paths of up to 32 steps. Even an all-diagonal path
// fits this cost budget, which bounds searches without a node-count cutoff.
const maxWalkPathCost = 32 * 14

func FindWalkPath(gat *res.GAT, fromX, fromY, toX, toY int) ([]worldstate.WalkStep, bool) {
	if gat == nil || !gat.InBounds(fromX, fromY) || !gat.InBounds(toX, toY) || !gat.Walkable(toX, toY) {
		return nil, false
	}
	start := PathPoint{X: fromX, Y: fromY}
	goal := PathPoint{X: toX, Y: toY}
	if PathHeuristic(start, goal) > maxWalkPathCost {
		return nil, false
	}
	open := &pathHeap{}
	heap.Init(open)
	startNode := &pathNode{point: start, g: 0, f: PathHeuristic(start, goal)}
	heap.Push(open, startNode)
	nodes := map[PathPoint]*pathNode{start: startNode}
	closed := make(map[PathPoint]struct{})

	for open.Len() > 0 {
		current := heap.Pop(open).(*pathNode)
		if _, ok := closed[current.point]; ok {
			continue
		}
		if current.point == goal {
			return reconstructPath(current), true
		}
		closed[current.point] = struct{}{}
		for _, next := range pathNeighbors(gat, current.point) {
			if _, ok := closed[next]; ok {
				continue
			}
			cost := current.g + PathStepCost(current.point, next)
			estimate := cost + PathHeuristic(next, goal)
			if estimate > maxWalkPathCost {
				continue
			}
			node, ok := nodes[next]
			if !ok {
				node = &pathNode{point: next, g: cost, f: estimate, parent: current}
				nodes[next] = node
				heap.Push(open, node)
				continue
			}
			if cost < node.g {
				node = &pathNode{point: next, g: cost, f: estimate, parent: current}
				nodes[next] = node
				heap.Push(open, node)
			}
		}
	}
	return nil, false
}

type PathPoint struct {
	X int
	Y int
}

type pathNode struct {
	point  PathPoint
	g      int
	f      int
	parent *pathNode
}

type pathHeap []*pathNode

func (h pathHeap) Len() int { return len(h) }

func (h pathHeap) Less(i, j int) bool {
	if h[i].f == h[j].f {
		return h[i].g > h[j].g
	}
	return h[i].f < h[j].f
}

func (h pathHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *pathHeap) Push(x any) { *h = append(*h, x.(*pathNode)) }

func (h *pathHeap) Pop() any {
	old := *h
	n := len(old)
	node := old[n-1]
	*h = old[:n-1]
	return node
}

func pathNeighbors(gat *res.GAT, point PathPoint) []PathPoint {
	var out []PathPoint
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			next := PathPoint{X: point.X + dx, Y: point.Y + dy}
			if !gat.InBounds(next.X, next.Y) {
				continue
			}
			if !gat.Walkable(next.X, next.Y) {
				continue
			}
			if dx != 0 && dy != 0 && (!gat.Walkable(point.X+dx, point.Y) || !gat.Walkable(point.X, point.Y+dy)) {
				continue
			}
			out = append(out, next)
		}
	}
	return out
}

func PathStepCost(from, to PathPoint) int {
	if from.X != to.X && from.Y != to.Y {
		return 14
	}
	return 10
}

func PathHeuristic(from, to PathPoint) int {
	dx := scene.AbsInt(to.X - from.X)
	dy := scene.AbsInt(to.Y - from.Y)
	if dx < dy {
		return 14*dx + 10*(dy-dx)
	}
	return 14*dy + 10*(dx-dy)
}

func reconstructPath(node *pathNode) []worldstate.WalkStep {
	var reversed []worldstate.WalkStep
	for node != nil {
		reversed = append(reversed, worldstate.WalkStep{X: node.point.X, Y: node.point.Y})
		node = node.parent
	}
	path := make([]worldstate.WalkStep, len(reversed))
	for i := range reversed {
		path[i] = reversed[len(reversed)-1-i]
	}
	return path
}
