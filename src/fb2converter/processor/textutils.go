package processor

import (
	"crypto/md5"
	"fmt"
	"io"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/beevik/etree"
)

const (
	strNBSP       = "\u00A0"
	strSOFTHYPHEN = "\u00AD"
)

type htmlHeader int

func (hl *htmlHeader) Inc() {
	if *hl < math.MaxInt32 {
		*hl++
	}
}

func (hl *htmlHeader) Dec() {
	if *hl > 0 {
		*hl--
	}
}

func (hl htmlHeader) Int() int {
	return int(hl)
}

func (hl htmlHeader) String(prefix string) string {
	if hl > 6 {
		hl = 6
	}
	return fmt.Sprintf("%s%d", prefix, hl)
}

// GetFirstRuneString returns first UTF-8 rune of the passed in string.
func GetFirstRuneString(in string) string {
	for _, c := range in {
		return string(c)
	}
	return ""
}

// GenSafeName takes a string and generates file name form it which is safe to use everywhere.
func GenSafeName(name string) string {
	h := md5.New()
	io.WriteString(h, name)
	return fmt.Sprintf("%x", h.Sum(nil))
}

var nameCleaner = strings.NewReplacer("\r", "", "\n", "", " ", "")

// SanitizeName in case name needs cleanup.
func SanitizeName(in string) (out string, changed bool) {
	out = nameCleaner.Replace(in)
	return out, out != in
}

var noteCleaner = regexp.MustCompile(`[\[{].*[\]}]`)

// SanitizeTitle removes footnote leftovers and CR (in case this is Windows).
func SanitizeTitle(in string) string {
	return strings.Replace(noteCleaner.ReplaceAllLiteralString(in, ""), "\r", "", -1)
}

// AllLines joins lines using space as a EOL replacement.
func AllLines(in string) string {
	return strings.Join(strings.Split(in, "\n"), " ")
}

// FirstLine returns first line for supplied string.
func FirstLine(in string) string {
	return strings.Split(in, "\n")[0]
}

// ReplaceKeywords scans provided string for keys from the map and replaces them with corresponding values from the map.
// Curly brackets '{' and '}' are special - they indicate conditional block. If all keys inside block were replaced with
// empty values - whole block inside curly brackets will be removed. Blocks could be nested. Curly brackets could be escaped
// with backslash if necessary.
func ReplaceKeywords(in string, m map[string]string) string {

	expandKeyword := func(in, key, value string) (string, bool) {
		if strings.Index(in, key) != -1 {
			return strings.Replace(in, key, value, -1), len(value) > 0
		}
		return in, false
	}

	expandAll := func(in string, m map[string]string) string {

		// NOTE: to provide stable results longer keywords should be replaced first (#authors then #author)
		var keys []string
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var expanded, ok bool
		for i := len(keys) - 1; i >= 0; i-- {
			in, ok = expandKeyword(in, keys[i], m[keys[i]])
			expanded = expanded || ok
		}
		if !expanded {
			return ""
		}
		return in
	}

	bopen, bclose := -1, -1
	for i, c := range in {
		if c == '{' && (i == 0 || i > 0 && in[i-1] != '\\') {
			bopen = i
		} else if c == '}' && (i == 0 || i > 0 && in[i-1] != '\\') {
			bclose = i
			break
		}
	}

	if bopen >= 0 && bclose > 0 && bopen < bclose {
		return ReplaceKeywords(in[:bopen]+expandAll(in[bopen+1:bclose], m)+in[bclose+1:], m)
	}
	return expandAll(in, m)
}

// CreateAuthorKeywordsMap prepares keywords map for replacement.
func CreateAuthorKeywordsMap(e *etree.Element) map[string]string {

	var f, m, l string

	rd := make(map[string]string)
	if n := e.SelectElement("first-name"); n != nil {
		if f = strings.TrimSpace(n.Text()); len(f) > 0 {
			rd["#f"], rd["#fi"] = f, GetFirstRuneString(f)+"."
		} else {
			rd["#f"], rd["#fi"] = "", ""
		}
	}
	if n := e.SelectElement("middle-name"); n != nil {
		if m = strings.TrimSpace(n.Text()); len(m) > 0 {
			rd["#m"], rd["#mi"] = m, GetFirstRuneString(m)+"."
		} else {
			rd["#m"], rd["#mi"] = "", ""
		}
	}
	if n := e.SelectElement("last-name"); n != nil {
		if l = strings.TrimSpace(n.Text()); len(l) > 0 {
			rd["#l"] = l
		} else {
			rd["#l"] = ""
		}
	}
	return rd
}

// CreateTitleKeywordsMap prepares keywords map for replacement.
func CreateTitleKeywordsMap(b *Book, pos int) map[string]string {
	rd := make(map[string]string)
	rd["#title"] = ""
	if len(b.Title) > 0 {
		rd["#title"] = b.Title
	}
	rd["#series"], rd["#abbrseries"] = "", ""
	if len(b.SeqName) > 0 {
		rd["#series"] = b.SeqName
		var abbr string
		for _, w := range strings.Split(b.SeqName, " ") {
			if len(w) > 0 {
				r, l := utf8.DecodeRuneInString(w)
				if r != utf8.RuneError && l > 0 {
					abbr += string(r)
				}
			}
		}
		if len(abbr) > 0 {
			rd["#abbrseries"] = strings.ToLower(abbr)
		}
	}
	rd["#number"], rd["#padnumber"] = "", ""
	if b.SeqNum > 0 {
		rd["#number"] = fmt.Sprintf("%d", b.SeqNum)
		rd["#padnumber"] = fmt.Sprintf(fmt.Sprintf("%%0%dd", pos), b.SeqNum)
	}
	rd["#date"] = ""
	if len(b.Date) > 0 {
		rd["#date"] = b.Date
	}
	return rd
}

// CreateFileNameKeywordsMap prepares keywords map for replacement.
func CreateFileNameKeywordsMap(b *Book, pos int) map[string]string {
	rd := make(map[string]string)
	rd["#title"] = ""
	if len(b.Title) > 0 {
		rd["#title"] = b.Title
	}
	rd["#series"], rd["#abbrseries"] = "", ""
	if len(b.SeqName) > 0 {
		rd["#series"] = b.SeqName
		var abbr string
		for _, w := range strings.Split(b.SeqName, " ") {
			if len(w) > 0 {
				r, l := utf8.DecodeRuneInString(w)
				if r != utf8.RuneError && l > 0 {
					abbr += string(r)
				}
			}
		}
		if len(abbr) > 0 {
			rd["#abbrseries"] = strings.ToLower(abbr)
		}
	}
	rd["#number"], rd["#padnumber"] = "", ""
	if b.SeqNum > 0 {
		rd["#number"] = fmt.Sprintf("%d", b.SeqNum)
		rd["#padnumber"] = fmt.Sprintf(fmt.Sprintf("%%0%dd", pos), b.SeqNum)
	}
	rd["#authors"] = b.BookAuthors(false)
	rd["#author"] = b.BookAuthors(true)
	rd["#bookid"] = b.ID.String()
	return rd
}

// AppendIfMissing well append string to slice only if it is not there already.
func AppendIfMissing(slice []string, str string) []string {
	for _, s := range slice {
		if s == str {
			return slice
		}
	}
	return append(slice, str)
}

// IsOneOf checks if string is present in slice of strings. Comparison is case insensitive.
func IsOneOf(name string, names []string) bool {
	for _, n := range names {
		if strings.EqualFold(name, n) {
			return true
		}
	}
	return false
}
