package tablewriter

import (
	"bytes"
	"flag"
	"io/ioutil"
	"testing"

	"cdr.dev/slog/sloggers/slogtest/assert"
)

var write = flag.Bool("write", false, "write to the golden files")

func TestTableWriter(t *testing.T) {
	type NestedRow struct {
		NestedOne string `table:"first_nested"`
		NestedTwo string `table:"second_nested"`
	}

	type Row struct {
		ID            string `table:"-"`
		Name          string
		BirthdayMonth int       `table:"birthday month"`
		Nested        NestedRow `table:"_"`
		Age           float32
	}

	items := []Row{
		{
			ID:            "13123lkjqlkj-2f323l--f23f",
			Name:          "Tom",
			BirthdayMonth: 12,
			Age:           28.12,
			Nested: NestedRow{
				NestedOne: "234-0934",
				NestedTwo: "2340-234234",
			},
		},
		{
			ID:            "afwaflkj23kl-2f323l--f23f",
			Name:          "Jerry",
			BirthdayMonth: 3,
			Age:           36.22,
			Nested: NestedRow{
				NestedOne: "aflfafe-afjlk",
				NestedTwo: "falj-fjlkjlkadf",
			},
		},
	}

	buf := bytes.NewBuffer(nil)
	err := WriteTable(buf, len(items), func(i int) interface{} { return items[i] })
	assert.Success(t, "write table", err)

	assertGolden(t, "table_output.golden", buf.Bytes())
}

func assertGolden(t *testing.T, path string, output []byte) {
	if *write {
		err := ioutil.WriteFile(path, output, 0777)
		assert.Success(t, "write file", err)
		return
	}
	goldenContent, err := ioutil.ReadFile(path)
	assert.Success(t, "read golden file", err)
	assert.Equal(t, "golden content matches", string(goldenContent), string(output))
}
