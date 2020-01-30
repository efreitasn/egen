package htmlp

import (
	"strconv"
	"testing"
)

func TestPretty(t *testing.T) {
	tests := []struct {
		in  []byte
		out []byte
		err error
	}{
		{
			[]byte(`<!DOCTYPE html><html lang="en"><head>  <meta charset="uft-8">  <meta name="viewport" content="width=device-width, initial-scale=1.0">  <meta http-equiv="X-UA-Compatible" content="ie=edge">  <title>Document</title></head><body><div>some</div><img src="foobar.png" alt="barfoo" /><div data-htmlp-ignore><span>aaa</span> <span>bbb</span></div></body></html>`),
			[]byte(`<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="uft-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
    <title>
      Document
    </title>
  </head>
  <body>
    <div>
      some
    </div>
    <img src="foobar.png" alt="barfoo" />
    <div><span>aaa</span> <span>bbb</span></div>
  </body>
</html>
`),
			nil,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res, err := Pretty(test.in)

			if err != test.err {
				t.Fatalf("got %v, want %v", err, test.err)
			}

			if string(res) != string(test.out) {
				t.Errorf("got %v, want %v", string(res), string(test.out))
			}
		})
	}
}
