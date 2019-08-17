package merkletree

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/dusk-network/dusk-crypto/hash"
)

type TestPayload struct {
	x string
}

func (t TestPayload) CalculateHash() ([]byte, error) {
	return hash.Sha3256([]byte(t.x))
}

/*
bellaHash := byte[]{
	123, 84, 169, 215, 192, 54, 108, 20, 6, 245, 41, 14, 121, 178, 145, 55, 94, 238, 113, 30, 4, 1, 80, 41, 151, 97, 73, 228, 46, 101, 217, 66
}

ciaoHash := byte[]{
	240, 108, 25, 222, 241, 183, 15, 22, 31, 89, 41, 192, 69, 200, 232, 63, 131, 145, 226, 85, 237, 211, 110, 40, 176, 99, 27, 26, 82, 181, 47, 28
}

bellaCiaoHash := byte[]{
	51, 101, 184, 162, 4, 3, 146, 102, 116, 250, 26, 17, 70, 19, 161, 207, 164, 139, 77, 55, 143, 221, 170, 91, 220, 236, 149, 89, 207, 220, 250, 249
}

ndoHash := byte[]{
	203, 213, 36, 240, 52, 96, 20, 37, 47, 174, 66, 155, 188, 249, 99, 254, 216, 245, 165, 217, 193, 27, 204, 39, 172, 133, 239, 199, 83, 79, 49, 46
}

scappiHash = byte[]{
	232, 252, 180, 188, 159, 124, 78, 237, 220, 96, 87, 100, 81, 105, 86, 36, 64, 130, 228, 6, 8, 122, 165, 167, 51, 238, 125, 227, 52, 106, 150, 28
}

ndoScappiHash = byte[]{
	235, 234, 254, 120, 164, 234, 222, 167, 17, 96, 71, 221, 85, 107, 106, 107, 26, 168, 214, 157, 164, 163, 180, 172, 35, 22, 255, 202, 186, 152, 212, 52
}

root = byte[]{
	88, 206, 79, 67, 184, 243, 45, 145, 95, 208, 173, 187, 93, 119, 173, 216, 206, 250, 233, 54, 65, 0, 166, 185, 102, 156, 49, 227, 107, 3, 178, 119
}
*/

type Fixture struct {
	Payloads     []Payload
	ExpectedHash []byte
}

var testTable = []Fixture{
	{
		Payloads: []Payload{
			TestPayload{
				x: "Bella",
			},
			TestPayload{
				x: "Ciao",
			},
			TestPayload{
				x: "Ndo",
			},
			TestPayload{
				x: "Scappi",
			},
		},
		ExpectedHash: []byte{
			88, 206, 79, 67, 184, 243, 45, 145, 95, 208, 173, 187, 93, 119, 173, 216, 206, 250, 233, 54, 65, 0, 166, 185, 102, 156, 49, 227, 107, 3, 178, 119,
		},
	},

	{
		Payloads: []Payload{
			TestPayload{
				x: "Hello",
			},
			TestPayload{
				x: "Hi",
			},
			TestPayload{
				x: "Hey",
			},
			TestPayload{
				x: "Hola",
			},
		},
		ExpectedHash: []byte{
			32, 188, 172, 153, 245, 171, 51, 156, 161, 201, 80, 58, 155, 97, 1, 79, 86, 175, 244, 91, 137, 105, 238, 155, 233, 126, 112, 151, 195, 101, 37, 220,
		},
	},
}

type uTest func(t *Tree, fixture Fixture, index int) error

func injectTest(t *testing.T, u uTest) {
	for i := 0; i < len(testTable); i++ {
		currentFixture := testTable[i]

		tree, err := NewTree(testTable[i].Payloads)
		if err != nil {
			t.Error("unexpected error: ", err)
		}

		if err := u(tree, currentFixture, i); err != nil {
			t.Errorf(err.Error())
		}

	}
}

var newTreeTest = func(tree *Tree, f Fixture, i int) error {
	eh := f.ExpectedHash
	root := tree.MerkleRoot

	if bytes.Compare(root, eh) != 0 {
		return fmt.Errorf("expecting hash %v but got %v", eh, root)
	}
	return nil
}

var merkleRootTest = func(tree *Tree, f Fixture, i int) error {
	root := tree.MerkleRoot
	if bytes.Compare(root, f.ExpectedHash) != 0 {
		return fmt.Errorf("expecting hash %v but got %v", f.ExpectedHash, root)
	}
	return nil
}

var rebuildTreeTest = func(tree *Tree, f Fixture, i int) error {
	root := tree.MerkleRoot
	if err := tree.RebuildTree(); err != nil {
		return err
	}

	if bytes.Compare(root, f.ExpectedHash) != 0 {
		return fmt.Errorf("expecting hash %v but got %v", f.ExpectedHash, root)
	}
	return nil
}

var verifyTree = func(tree *Tree, f Fixture, i int) error {

	v1, err := VerifyTree(tree)
	if err != nil {
		return err
	}

	if !v1 {
		return errors.New("expected tree to be valid")
	}

	tree.Root.Hash = []byte{1}
	tree.MerkleRoot = []byte{1}

	v2, err := VerifyTree(tree)
	if err != nil {
		return err
	}

	if v2 {
		return errors.New("expected tree to be invalid")
	}
	return nil
}

var verifyContent = func(tree *Tree, f Fixture, i int) error {

	for _, payload := range f.Payloads {
		v, err := tree.VerifyContent(payload)
		if err != nil {
			return err
		}
		if !v {
			return fmt.Errorf("encountered invalid payload %s", payload)
		}
	}

	v, err := tree.VerifyContent(TestPayload{x: "NotInTestTable"})
	if err != nil {
		return err
	}

	if v {
		return errors.New("verification should have failed")
	}

	return nil
}

func TestMerkleTree_RebuildWithPayload(t *testing.T) {
	for i := 0; i < len(testTable)-1; i++ {
		tree, err := NewTree(testTable[i].Payloads)
		if err != nil {
			t.Error("unexpected error:  ", err)
		}
		err = tree.RebuildTreeUsing(testTable[i+1].Payloads)
		if err != nil {
			t.Error("unexpected error:  ", err)
		}
		if bytes.Compare(tree.MerkleRoot, testTable[i+1].ExpectedHash) != 0 {
			t.Errorf("expected hash equal to %v got %v", testTable[i+1].ExpectedHash, tree.MerkleRoot)
		}
	}
}

func TestSuite(t *testing.T) {
	injectTest(t, newTreeTest)
	injectTest(t, merkleRootTest)
	injectTest(t, rebuildTreeTest)
	t.Run("Rebuilding the MerkleTree with the payload yields the same MerkleTree", TestMerkleTree_RebuildWithPayload)
	injectTest(t, verifyTree)
	injectTest(t, verifyContent)
}
