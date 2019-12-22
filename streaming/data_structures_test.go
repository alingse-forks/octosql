package streaming

import (
	"log"
	"testing"

	"github.com/cube2222/octosql"
	"github.com/dgraph-io/badger/v2"
	"github.com/pkg/errors"
)

func TestLinkedList(t *testing.T) {
	prefix := "test_linked_list"
	db, err := badger.Open(badger.DefaultOptions("test"))
	if err != nil {
		log.Fatal(err)
	}

	defer db.DropAll()

	store := NewBadgerStorage(db)
	txn := store.BeginTransaction().WithPrefix([]byte(prefix))

	linkedList := NewLinkedList(txn)

	values := []octosql.Value{
		octosql.MakeInt(1),
		octosql.MakeInt(2),
		octosql.MakeInt(3),
		octosql.MakeInt(4),
		octosql.MakeInt(5),
	}

	for i := 0; i < len(values); i++ {
		err := linkedList.Append(&values[i])
		if err != nil {
			log.Fatal(err)
		}
	}

	/* test if all values are there */
	iter := linkedList.GetIterator()
	areEqual, err := testIterator(iter, values)
	if err != nil {
		log.Fatal(err)
	}

	if !areEqual {
		log.Fatal("The iterator doesn't contain the expected values")
	}

	/* test peek */
	var value octosql.Value

	err = linkedList.Peek(&value)
	if err != nil {
		log.Fatal(err)
	}

	if !octosql.AreEqual(value, values[0]) {
		log.Fatal("the value returned by Peek() isn't the first value inserted")
	}

	err = linkedList.Peek(&value) //Peek shouldn't modify the linkedList in any way
	if err != nil {
		log.Fatal(err)
	}

	if !octosql.AreEqual(value, values[0]) {
		log.Fatal("the value returned by Peek() the second time isn't the first value inserted")
	}

	/* test pop */
	err = linkedList.Pop(&value)
	if err != nil {
		log.Fatal(err)
	}

	if !octosql.AreEqual(value, values[0]) {
		log.Fatal("the value returned by Pop() isn't the first value inserted")
	}

	_ = iter.Close() //we need to close the iterator, to be able to get the next one

	iter = linkedList.GetIterator()
	areEqual, err = testIterator(iter, values[1:])

	if err != nil {
		log.Fatal(err)
	}

	if !areEqual {
		log.Fatal("The iterator doesn't contain the expected values")
	}

	/* test pop again but this time create a new Linked List to operate on the same data*/
	linkedList2 := NewLinkedList(txn)

	err = linkedList2.Pop(&value)
	if err != nil {
		log.Fatal(err)
	}

	if !octosql.AreEqual(value, values[1]) {
		log.Fatal("the value returned by Pop() isn't the first value inserted")
	}

	_ = iter.Close() //we need to close the iterator, to be able to get the next one

	iter = linkedList2.GetIterator()
	areEqual, err = testIterator(iter, values[2:])
	_ = iter.Close()

	if err != nil {
		log.Fatal(err)
	}

	if !areEqual {
		log.Fatal("The iterator doesn't contain the expected values")
	}

	/* test clear */
	err = linkedList2.Clear()
	if err != nil {
		log.Fatal(err)
	}

	/* test if linked list is actually empty */
	iter = linkedList2.GetIterator()
	areEqual, err = testIterator(iter, []octosql.Value{})
	_ = iter.Close()

	if err != nil {
		log.Fatal(err)
	}

	if !areEqual {
		log.Fatal("The iterator should be empty")
	}

	_, err = txn.Get(linkedListLengthKey)
	if err != badger.ErrKeyNotFound {
		log.Fatal("the linked list length element index should be empty")
	}

	_, err = txn.Get(linkedListFirstElementKey)
	if err != badger.ErrKeyNotFound {
		log.Fatal("the linked list first element should be empty")
	}

	/* we should still be able to append elements tho */
	err = linkedList2.Append(&values[0])
	if err != nil {
		log.Fatal(err)
	}

	iter = linkedList2.GetIterator()
	areEqual, err = testIterator(iter, values[:1])
	_ = iter.Close()

	if err != nil {
		log.Fatal(err)
	}

	if !areEqual {
		log.Fatal("The iterator doesn't contain the expected values")
	}

	err = linkedList2.Clear()
	if err != nil {
		log.Fatal(err)
	}
}

func TestMap(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("test"))
	if err != nil {
		log.Fatal(err)
	}

	defer db.DropAll()

	store := NewBadgerStorage(db)
	txn := store.BeginTransaction()

	badgerMap := NewMap(txn.WithPrefix([]byte("map_prefix_")))

	key1 := octosql.MakeString("aaa")
	value1 := octosql.MakeString("siemanko")
	err = badgerMap.Set(&key1, &value1)
	if err != nil {
		panic(err)
	}

	key2 := octosql.MakeString("bbb")
	value2 := octosql.MakeString("eluwina")
	err = badgerMap.Set(&key2, &value2)
	if err != nil {
		panic(err)
	}

	it := badgerMap.GetIterator()

	var key octosql.Value
	var val octosql.Value

	for {
		err = it.Next(&key, &val)
		if err == ErrEndOfIterator {
			return
		} else if err != nil {
			log.Fatal(err)
		}

		println(key.AsString(), val.AsString())
	}

}

func testIterator(iter SimpleIterator, expectedValues []octosql.Value) (bool, error) {
	var value octosql.Value

	for i := 0; i < len(expectedValues); i++ {
		err := iter.Next(&value)

		if err != nil {
			return false, errors.Wrap(err, "expected a value, got an error")
		}

		if !octosql.AreEqual(value, expectedValues[i]) {
			return false, errors.Errorf("mismatch of values at index %d", i)
		}
	}

	err := iter.Next(&value)
	if err != ErrEndOfIterator {
		return false, errors.New("expected ErrEndOfStream")
	}

	return true, nil
}
