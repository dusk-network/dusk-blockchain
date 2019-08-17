package merkletree

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-crypto/hash"
)

// Payload is data that can be stored and checked in the Merkletree.
// Types implementing this interface can be items of the Merkletree
type Payload interface {
	CalculateHash() ([]byte, error)
}

// Tree struct representing the merkletree data structure.
// It includes a pointer to the Root of the tree, list of pointers to the Leaves and the MerkleRoot
type Tree struct {
	Root       *Node
	MerkleRoot []byte
	Leaves     []*Node
}

// Node of the Merkletree which includes pointers to the Parent, the immediate Left and Right children
// and whether it is a leaf or not
type Node struct {
	Parent *Node
	Left   *Node
	Right  *Node
	Hash   []byte
	Data   Payload
	IsLeaf bool
	IsDup  bool
}

// VerifyNode calculates the hash of each level until hitting a leaf.
// It returns the Hash of Node n
func VerifyNode(n *Node) ([]byte, error) {
	if n.IsLeaf {
		return n.Data.CalculateHash()
	}

	rightBytes, err := VerifyNode(n.Right)
	if err != nil {
		return nil, err
	}

	leftBytes, err := VerifyNode(n.Left)
	if err != nil {
		return nil, err
	}

	return hash.Sha3256(append(leftBytes, rightBytes...))
}

// CalculateNodeHash is a helper function for calculating the hash of a node
func CalculateNodeHash(n *Node) ([]byte, error) {
	if n.IsLeaf {
		return n.Data.CalculateHash()
	}

	return hash.Sha3256(append(n.Left.Hash, n.Right.Hash...))
}

// NewTree constructs a merkletree using the Payload/Content pl
func NewTree(pl []Payload) (*Tree, error) {
	root, leaves, err := create(pl)
	if err != nil {
		return nil, err
	}

	return &Tree{
		Root:       root,
		MerkleRoot: root.Hash,
		Leaves:     leaves,
	}, nil
}

// create is the helper to map the Payload array recursively into a Hash tree with a root and a set of leaf nodes
// returns an error if the Payload array is empty
func create(pl []Payload) (*Node, []*Node, error) {
	if len(pl) == 0 {
		return nil, nil, errors.New("cannot construct tree with no content")
	}

	var leaves []*Node
	for _, payload := range pl {
		h, err := payload.CalculateHash()
		if err != nil {
			return nil, nil, err
		}

		leaves = append(leaves, &Node{
			Data:   payload,
			Hash:   h,
			IsLeaf: true,
		})
	}

	if len(leaves)%2 == 1 {
		previous := leaves[len(leaves)-1]
		duplicate := &Node{
			Hash:   previous.Hash,
			Data:   previous.Data,
			IsLeaf: true,
			IsDup:  true,
		}
		leaves = append(leaves, duplicate)
	}

	root, err := createIntermediate(leaves)
	if err != nil {
		return nil, nil, err
	}
	return root, leaves, nil
}

// createIntermediate is the helper function that for a set of leafs nodes, recursively builds the intermediate and root level of a Tree. Returns the resulting root node
func createIntermediate(nl []*Node) (*Node, error) {
	var nodes []*Node
	for i := 0; i < len(nl); i += 2 {
		var left, right int = i, i + 1
		if i+1 == len(nl) {
			right = i
		}
		payloadHash := append(nl[left].Hash, nl[right].Hash...)

		h, err := hash.Sha3256(payloadHash)
		if err != nil {
			return nil, err
		}

		node := &Node{
			Left:  nl[left],
			Right: nl[right],
			Hash:  h,
		}

		nodes = append(nodes, node)
		nl[left].Parent = node
		nl[right].Parent = node

		if len(nl) == 2 {
			return node, nil
		}
	}

	return createIntermediate(nodes)
}

// RebuildTree is a helper method to rebuild a Tree with only the Data contained in the leafs
func (t *Tree) RebuildTree() error {
	var pl []Payload
	for _, node := range t.Leaves {
		pl = append(pl, node.Data)
	}

	return t.RebuildTreeUsing(pl)
}

// RebuildTreeUsing is a helper method to replace the Data in the Markletree and rebuild the Tree entirely
// The root gets replaced but the Merkletree survives the operation
// Returns an error if there is no payload
func (t *Tree) RebuildTreeUsing(pl []Payload) error {
	root, leaves, err := create(pl)
	if err != nil {
		return err
	}

	t.Root = root
	t.Leaves = leaves
	t.MerkleRoot = root.Hash
	return nil
}

// VerifyTree Verifies each Node's hash and check if the resulted root hash is correct compared to the reported root hash of the merkle tree
func VerifyTree(t *Tree) (bool, error) {
	calculatedRoot, err := VerifyNode(t.Root)
	if err != nil {
		return false, err
	}

	if bytes.Compare(t.MerkleRoot, calculatedRoot) == 0 {
		return true, nil
	}

	return false, nil
}

//VerifyContent verifies whether a piece of data is in the merkle tree
func (t *Tree) VerifyContent(data Payload) (bool, error) {
	for _, leaf := range t.Leaves {
		ok, err := equals(leaf.Data, data)
		if err != nil {
			return false, err
		}

		if ok {
			currentParent := leaf.Parent
			for currentParent != nil {
				rightBytes, err := CalculateNodeHash(currentParent.Right)
				if err != nil {
					return false, err
				}

				leftBytes, err := CalculateNodeHash(currentParent.Left)
				if err != nil {
					return false, err
				}

				h, err := hash.Sha3256(append(leftBytes, rightBytes...))

				if err != nil {
					return false, err
				}

				if bytes.Compare(h, currentParent.Hash) != 0 {
					return false, nil
				}

				currentParent = currentParent.Parent
			}

			return true, nil
		}
	}

	return false, nil
}

// equals returns true, if two payloads hash to the same value
func equals(a, b Payload) (bool, error) {
	hashA, err := a.CalculateHash()
	if err != nil {
		return false, err
	}
	hashB, err := b.CalculateHash()
	if err != nil {
		return false, err
	}

	return bytes.Equal(hashA, hashB), nil
}
