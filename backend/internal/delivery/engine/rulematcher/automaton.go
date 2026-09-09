package rulematcher

import dto "dpdp-backend/internal/delivery/engine/dto/evaluation"

const rootNode = 0

type node struct {
	next    map[byte]int
	fail    int
	lengths []int
	outputs []int
}

type Automaton struct {
	nodes []node
}

func BuildAutomaton(patterns []string) *Automaton {
	automaton := &Automaton{nodes: []node{{next: map[byte]int{}}}}

	for index, pattern := range patterns {
		if pattern == "" {
			continue
		}

		current := rootNode

		for position := range len(pattern) {
			character := pattern[position]

			following, ok := automaton.nodes[current].next[character]
			if !ok {
				automaton.nodes = append(automaton.nodes, node{next: map[byte]int{}})
				following = len(automaton.nodes) - 1
				automaton.nodes[current].next[character] = following
			}

			current = following
		}

		automaton.nodes[current].outputs = append(automaton.nodes[current].outputs, index)
		automaton.nodes[current].lengths = append(automaton.nodes[current].lengths, len(pattern))
	}

	automaton.link()

	return automaton
}

func (a *Automaton) link() {
	queue := make([]int, 0, len(a.nodes))

	for character, following := range a.nodes[rootNode].next {
		a.nodes[following].fail = rootNode
		queue = append(queue, following)

		_ = character
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for character, following := range a.nodes[current].next {
			fallback := a.nodes[current].fail

			for fallback != rootNode {
				if _, ok := a.nodes[fallback].next[character]; ok {
					break
				}

				fallback = a.nodes[fallback].fail
			}

			if target, ok := a.nodes[fallback].next[character]; ok && target != following {
				a.nodes[following].fail = target
			} else {
				a.nodes[following].fail = rootNode
			}

			suffix := a.nodes[following].fail
			a.nodes[following].outputs = append(a.nodes[following].outputs, a.nodes[suffix].outputs...)
			a.nodes[following].lengths = append(a.nodes[following].lengths, a.nodes[suffix].lengths...)

			queue = append(queue, following)
		}
	}
}

func (a *Automaton) Find(text string) []dto.AutomatonHit {
	hits := make([]dto.AutomatonHit, 0)
	current := rootNode

	for position := range len(text) {
		character := text[position]

		for {
			if following, ok := a.nodes[current].next[character]; ok {
				current = following
				break
			}

			if current == rootNode {
				break
			}

			current = a.nodes[current].fail
		}

		for index, pattern := range a.nodes[current].outputs {
			length := a.nodes[current].lengths[index]

			hits = append(hits, dto.AutomatonHit{
				Index: pattern,
				Start: position - length + 1,
				End:   position + 1,
			})
		}
	}

	return hits
}
