package maps

import (
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
	open := pathHeap{}
	startNode := &pathNode{point: start, f: pathPriority(start, goal)}
	open.push(startNode)
	nodes := map[PathPoint]*pathNode{start: startNode}

	for len(open) > 0 {
		current := open.pop()
		if current.point == goal {
			return reconstructPath(current), true
		}
		for _, next := range pathNeighbors(gat, current.point) {
			cost := current.g + PathStepCost(current.point, next)
			if cost+PathHeuristic(next, goal) > maxWalkPathCost {
				continue
			}
			node, ok := nodes[next]
			if !ok {
				node = &pathNode{point: next, index: -1}
				nodes[next] = node
			} else if cost >= node.g {
				continue
			}
			node.g, node.f, node.parent = cost, cost+pathPriority(next, goal), current
			if !ok {
				open.push(node)
			} else if node.index >= 0 {
				// Like the classic client, only reprioritize cells still open.
				open.up(node.index)
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
	index  int
}

// Match the classic client's heap: preserve ties on insertion, prefer the
// right child on removal, then sift the replacement back up from the leaf.
// A different tie order can draw a different route from the server's.
type pathHeap []*pathNode

func (h pathHeap) swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index, h[j].index = i, j
}

func (h pathHeap) up(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if h[parent].f <= h[i].f {
			break
		}
		h.swap(i, parent)
		i = parent
	}
}

func (h *pathHeap) push(node *pathNode) {
	node.index = len(*h)
	*h = append(*h, node)
	h.up(node.index)
}

func (h *pathHeap) pop() *pathNode {
	node := (*h)[0]
	last := len(*h) - 1
	h.swap(0, last)
	(*h)[last] = nil
	*h = (*h)[:last]
	node.index = -1
	i := 0
	for child := 1; child < len(*h); child = 2*i + 1 {
		if child+1 < len(*h) && (*h)[child+1].f <= (*h)[child].f {
			child++
		}
		h.swap(i, child)
		i = child
	}
	h.up(i)
	return node
}

func pathNeighbors(gat *res.GAT, point PathPoint) []PathPoint {
	var out []PathPoint
	// The order is significant when routes have the same priority.
	for _, delta := range [...]PathPoint{{1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}, {0, -1}} {
		dx, dy := delta.X, delta.Y
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
	return out
}

func pathPriority(from, to PathPoint) int {
	return 10 * (scene.AbsInt(to.X-from.X) + scene.AbsInt(to.Y-from.Y))
}

func PathStepCost(from, to PathPoint) int {
	if from.X != to.X && from.Y != to.Y {
		return 14
	}
	return 10
}

// PathHeuristic is a lower bound for search limits and approach-cell scoring.
// The classic client's route priority deliberately uses Manhattan distance.
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
