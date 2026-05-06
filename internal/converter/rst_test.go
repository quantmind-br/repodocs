package converter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertRST_Headings(t *testing.T) {
	input := `Top Level
=========

Subsection
----------

Detail
~~~~~~

Body text.
`
	out, err := ConvertRST([]byte(input))
	require.NoError(t, err)
	got := string(out)

	assert.Contains(t, got, "# Top Level")
	assert.Contains(t, got, "## Subsection")
	assert.Contains(t, got, "### Detail")
	assert.Contains(t, got, "Body text.")
	assert.NotContains(t, got, "=========")
	assert.NotContains(t, got, "~~~~~~")
}

func TestConvertRST_HeadingOverAndUnder(t *testing.T) {
	input := `=========
Title One
=========

Body.
`
	out, err := ConvertRST([]byte(input))
	require.NoError(t, err)
	got := string(out)
	assert.Contains(t, got, "# Title One")
	assert.NotContains(t, got, "=========")
}

func TestConvertRST_CodeBlock(t *testing.T) {
	input := `Intro paragraph.

.. code-block:: python

   def hello():
       return "world"

Outro.
`
	out, err := ConvertRST([]byte(input))
	require.NoError(t, err)
	got := string(out)
	assert.Contains(t, got, "```python")
	assert.Contains(t, got, "def hello():")
	assert.Contains(t, got, `    return "world"`)
	assert.Contains(t, got, "```\n")
	assert.NotContains(t, got, ".. code-block::")
}

func TestConvertRST_CodeBlockWithOptions(t *testing.T) {
	input := `.. code-block:: go
   :linenos:
   :emphasize-lines: 1

   package main

   func main() {}
`
	out, err := ConvertRST([]byte(input))
	require.NoError(t, err)
	got := string(out)
	assert.Contains(t, got, "```go")
	assert.Contains(t, got, "package main")
	assert.NotContains(t, got, ":linenos:")
	assert.NotContains(t, got, ":emphasize-lines:")
}

func TestConvertRST_LiteralBlock(t *testing.T) {
	input := `Look at this::

   raw literal
   with two lines

After.
`
	out, err := ConvertRST([]byte(input))
	require.NoError(t, err)
	got := string(out)
	assert.Contains(t, got, "Look at this:")
	assert.NotContains(t, got, "Look at this::")
	assert.Contains(t, got, "```\nraw literal\nwith two lines\n```")
	assert.Contains(t, got, "After.")
}

func TestConvertRST_Admonitions(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		wanted string
	}{
		{"note", ".. note::\n\n   Heads up.\n", "> [!NOTE]"},
		{"warning", ".. warning::\n\n   Careful!\n", "> [!WARNING]"},
		{"tip", ".. tip::\n\n   Pro tip.\n", "> [!TIP]"},
		{"important", ".. important::\n\n   Critical.\n", "> [!IMPORTANT]"},
		{"danger maps to warning", ".. danger::\n\n   Boom.\n", "> [!WARNING]"},
		{"hint maps to tip", ".. hint::\n\n   Hi.\n", "> [!TIP]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := ConvertRST([]byte(tc.input))
			require.NoError(t, err)
			got := string(out)
			assert.Contains(t, got, tc.wanted)
			assert.NotContains(t, got, ".. ")
		})
	}
}

func TestConvertRST_AdmonitionInlineFirstLine(t *testing.T) {
	input := `.. note:: This is on the directive line.

   And continues here.
`
	out, err := ConvertRST([]byte(input))
	require.NoError(t, err)
	got := string(out)
	assert.Contains(t, got, "> [!NOTE]")
	assert.Contains(t, got, "> This is on the directive line.")
	assert.Contains(t, got, "> And continues here.")
}

func TestConvertRST_BulletList(t *testing.T) {
	input := `* one
* two
* three
`
	out, err := ConvertRST([]byte(input))
	require.NoError(t, err)
	got := string(out)
	assert.Contains(t, got, "- one")
	assert.Contains(t, got, "- two")
	assert.Contains(t, got, "- three")
}

func TestConvertRST_EnumeratedList(t *testing.T) {
	input := `1. first
2. second
#. third
`
	out, err := ConvertRST([]byte(input))
	require.NoError(t, err)
	got := string(out)
	assert.Contains(t, got, "1. first")
	assert.Contains(t, got, "2. second")
	assert.Contains(t, got, "1. third")
}

func TestConvertRST_InlineLiteral(t *testing.T) {
	out, err := ConvertRST([]byte("Use ``foo()`` for things.\n"))
	require.NoError(t, err)
	got := string(out)
	assert.Contains(t, got, "Use `foo()` for things.")
	assert.NotContains(t, got, "``")
}

func TestConvertRST_Hyperlink(t *testing.T) {
	out, err := ConvertRST([]byte("See `Python <https://python.org>`_ docs.\n"))
	require.NoError(t, err)
	got := string(out)
	assert.Contains(t, got, "[Python](https://python.org)")
}

func TestConvertRST_Role(t *testing.T) {
	out, err := ConvertRST([]byte("Call :func:`my_function` to start.\n"))
	require.NoError(t, err)
	got := string(out)
	assert.Contains(t, got, "Call my_function to start.")
	assert.NotContains(t, got, ":func:")
}

func TestConvertRST_AnonymousReference(t *testing.T) {
	out, err := ConvertRST([]byte("See `Section Title`_ below.\n"))
	require.NoError(t, err)
	got := string(out)
	assert.Contains(t, got, "See Section Title below.")
	assert.NotContains(t, got, "`_")
}

func TestConvertRST_Image(t *testing.T) {
	input := `.. image:: ./diagrams/flow.png
`
	out, err := ConvertRST([]byte(input))
	require.NoError(t, err)
	got := string(out)
	assert.Contains(t, got, "![flow](./diagrams/flow.png)")
}

func TestConvertRST_Figure(t *testing.T) {
	input := `.. figure:: img/logo.svg
   :alt: Project logo
   :width: 200
`
	out, err := ConvertRST([]byte(input))
	require.NoError(t, err)
	got := string(out)
	assert.Contains(t, got, "![Project logo](img/logo.svg)")
}

func TestConvertRST_DropDirectives(t *testing.T) {
	input := `.. toctree::
   :maxdepth: 2

   intro
   guide
   api

Real content.

.. autodoc::
   module

More content.
`
	out, err := ConvertRST([]byte(input))
	require.NoError(t, err)
	got := string(out)
	assert.NotContains(t, got, "toctree")
	assert.NotContains(t, got, "autodoc")
	assert.NotContains(t, got, ":maxdepth:")
	assert.Contains(t, got, "Real content.")
	assert.Contains(t, got, "More content.")
}

func TestConvertRST_DropLinkTargets(t *testing.T) {
	input := `.. _label: https://example.com

Body text.
`
	out, err := ConvertRST([]byte(input))
	require.NoError(t, err)
	got := string(out)
	assert.NotContains(t, got, "_label:")
	assert.Contains(t, got, "Body text.")
}

func TestConvertRST_FallbackPlainText(t *testing.T) {
	input := `Just a plain paragraph.

Another one with **strong** and *emphasis*.
`
	out, err := ConvertRST([]byte(input))
	require.NoError(t, err)
	got := string(out)
	assert.Contains(t, got, "Just a plain paragraph.")
	assert.Contains(t, got, "**strong**")
	assert.Contains(t, got, "*emphasis*")
}

func TestConvertRST_EmptyInput(t *testing.T) {
	out, err := ConvertRST([]byte(""))
	require.NoError(t, err)
	assert.Empty(t, strings.TrimSpace(string(out)))
}

func TestConvertRST_CRLFNormalization(t *testing.T) {
	input := "Title\r\n=====\r\n\r\nBody.\r\n"
	out, err := ConvertRST([]byte(input))
	require.NoError(t, err)
	got := string(out)
	assert.Contains(t, got, "# Title")
	assert.Contains(t, got, "Body.")
	assert.NotContains(t, got, "\r")
}

func TestConvertRST_HeadingLevelsByOrder(t *testing.T) {
	// Per RST, the order in which adornment characters first appear defines
	// the level. `=` then `-` then `~` -> H1, H2, H3.
	input := `Alpha
=====

Beta
----

Gamma
~~~~~
`
	out, err := ConvertRST([]byte(input))
	require.NoError(t, err)
	got := string(out)
	idxAlpha := strings.Index(got, "# Alpha")
	idxBeta := strings.Index(got, "## Beta")
	idxGamma := strings.Index(got, "### Gamma")
	assert.Greater(t, idxAlpha, -1)
	assert.Greater(t, idxBeta, idxAlpha)
	assert.Greater(t, idxGamma, idxBeta)
}

func TestConvertRST_NoExtraBlankLines(t *testing.T) {
	input := `Title
=====


Body.
`
	out, err := ConvertRST([]byte(input))
	require.NoError(t, err)
	got := string(out)
	// At most one blank line between blocks.
	assert.NotContains(t, got, "\n\n\n")
}
