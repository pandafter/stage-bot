package voice

import (
	"regexp"
	"strings"
	"unicode"
)

// SanitizeForTTS rewrites bot text so ElevenLabs pronounces it naturally.
// It strips emojis, converts prices and abbreviations to spoken form, and
// normalizes punctuation that the model tends to read literally.
func SanitizeForTTS(text string) string {
	s := text

	s = replaceURLs(s)
	s = replacePrices(s)
	s = replaceTimes(s)
	s = replaceAbbreviations(s)
	s = replaceCommonSymbols(s)
	s = stripEmojis(s)
	s = collapseWhitespace(s)

	return strings.TrimSpace(s)
}

var (
	reURL        = regexp.MustCompile(`https?://\S+|www\.\S+`)
	rePrice      = regexp.MustCompile(`\$\s*([\d]{1,3}(?:[.,]\d{3})+|\d+)`)
	reTime       = regexp.MustCompile(`\b(\d{1,2}):(\d{2})\b`)
	reMultiSpace = regexp.MustCompile(`\s+`)
)

func replaceURLs(s string) string {
	return reURL.ReplaceAllString(s, "el link que te paso abajo")
}

// replacePrices converts "$890.000" into "ochocientos noventa mil pesos".
func replacePrices(s string) string {
	return rePrice.ReplaceAllStringFunc(s, func(m string) string {
		digits := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, m)
		if digits == "" {
			return m
		}
		return spellNumberCOP(digits) + " pesos"
	})
}

// replaceTimes converts "3:30" into "tres y media", "9:00" into "nueve en punto".
// Falls back to reading digits when minutes are irregular.
func replaceTimes(s string) string {
	return reTime.ReplaceAllStringFunc(s, func(m string) string {
		parts := reTime.FindStringSubmatch(m)
		if len(parts) != 3 {
			return m
		}
		h := parts[1]
		min := parts[2]
		hourWord := spellNumberCOP(h)
		switch min {
		case "00":
			return hourWord + " en punto"
		case "30":
			return hourWord + " y media"
		case "15":
			return hourWord + " y cuarto"
		case "45":
			return hourWord + " y cuarenta y cinco"
		default:
			return hourWord + " y " + spellNumberCOP(min)
		}
	})
}

var abbreviations = map[string]string{
	"Sr.":    "Señor",
	"Sra.":   "Señora",
	"Dr.":    "Doctor",
	"Dra.":   "Doctora",
	"Ing.":   "Ingeniero",
	"Av.":    "Avenida",
	"Cra.":   "Carrera",
	"Cl.":    "Calle",
	"Kr.":    "Carrera",
	"km":     "kilómetros",
	"kms":    "kilómetros",
	"hrs":    "horas",
	"hr":     "hora",
	"mins":   "minutos",
	"min":    "minutos",
	"nro.":   "número",
	"No.":    "número",
	"Nº":     "número",
	"vs.":    "versus",
	"aprox.": "aproximadamente",
	"etc.":   "etcétera",
}

func replaceAbbreviations(s string) string {
	for abbr, full := range abbreviations {
		s = strings.ReplaceAll(s, " "+abbr+" ", " "+full+" ")
		if strings.HasPrefix(s, abbr+" ") {
			s = full + " " + strings.TrimPrefix(s, abbr+" ")
		}
		if strings.HasSuffix(s, " "+abbr) {
			s = strings.TrimSuffix(s, " "+abbr) + " " + full
		}
	}
	return s
}

func replaceCommonSymbols(s string) string {
	replacements := map[string]string{
		"—":  ",",
		"–":  ",",
		"…":  ".",
		" - ": ", ",
		"%":  " por ciento",
		"&":  " y ",
		"/":  " o ",
		"(":  ", ",
		")":  ", ",
		"\"": "",
		"“":  "",
		"”":  "",
		"«":  "",
		"»":  "",
		"*":  "",
		"_":  " ",
		"#":  "",
	}
	for old, new := range replacements {
		s = strings.ReplaceAll(s, old, new)
	}
	s = collapseRepeatedPunct(s)
	return s
}

// collapseRepeatedPunct turns "!!!" into "!" and ",,," into ",".
func collapseRepeatedPunct(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	var prev rune
	for _, r := range s {
		if isCollapsiblePunct(r) && r == prev {
			continue
		}
		b.WriteRune(r)
		prev = r
	}
	return b.String()
}

func isCollapsiblePunct(r rune) bool {
	return r == '.' || r == ',' || r == '!' || r == '?'
}

func stripEmojis(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if isEmoji(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// isEmoji reports whether the rune is a pictograph, symbol or variation
// selector that ElevenLabs would read literally (e.g. "carita feliz").
func isEmoji(r rune) bool {
	switch {
	case r >= 0x1F300 && r <= 0x1FAFF: // misc symbols, pictographs, emoji
		return true
	case r >= 0x2600 && r <= 0x27BF: // misc symbols, dingbats
		return true
	case r >= 0x1F000 && r <= 0x1F2FF: // mahjong, cards, enclosed
		return true
	case r == 0xFE0F || r == 0x200D: // variation selector / ZWJ
		return true
	case unicode.Is(unicode.So, r): // "Symbol, Other"
		return true
	}
	return false
}

func collapseWhitespace(s string) string {
	return reMultiSpace.ReplaceAllString(s, " ")
}

// spellNumberCOP spells out a positive integer in Colombian Spanish.
// Handles up to billions; good enough for prices like 890.000 or 1.500.000.
func spellNumberCOP(digits string) string {
	digits = strings.TrimLeft(digits, "0")
	if digits == "" {
		return "cero"
	}
	n := 0
	for _, c := range digits {
		n = n*10 + int(c-'0')
	}
	return spellInt(n)
}

func spellInt(n int) string {
	if n == 0 {
		return "cero"
	}
	if n < 0 {
		return "menos " + spellInt(-n)
	}

	var parts []string
	if n >= 1_000_000_000 {
		billions := n / 1_000_000_000
		if billions == 1 {
			parts = append(parts, "mil millones")
		} else {
			parts = append(parts, spellInt(billions)+" mil millones")
		}
		n %= 1_000_000_000
	}
	if n >= 1_000_000 {
		millions := n / 1_000_000
		if millions == 1 {
			parts = append(parts, "un millón")
		} else {
			parts = append(parts, spellInt(millions)+" millones")
		}
		n %= 1_000_000
	}
	if n >= 1_000 {
		thousands := n / 1_000
		if thousands == 1 {
			parts = append(parts, "mil")
		} else {
			parts = append(parts, spellInt(thousands)+" mil")
		}
		n %= 1_000
	}
	if n > 0 {
		parts = append(parts, spellUnder1000(n))
	}
	return strings.Join(parts, " ")
}

func spellUnder1000(n int) string {
	if n == 100 {
		return "cien"
	}
	if n < 30 {
		return under30[n]
	}

	var parts []string
	if n >= 100 {
		parts = append(parts, hundreds[n/100])
		n %= 100
	}
	if n == 0 {
		return strings.Join(parts, " ")
	}
	if n < 30 {
		parts = append(parts, under30[n])
		return strings.Join(parts, " ")
	}
	tens := n / 10
	ones := n % 10
	if ones == 0 {
		parts = append(parts, tensWord[tens])
	} else {
		parts = append(parts, tensWord[tens]+" y "+under30[ones])
	}
	return strings.Join(parts, " ")
}

var under30 = []string{
	"", "uno", "dos", "tres", "cuatro", "cinco", "seis", "siete", "ocho", "nueve",
	"diez", "once", "doce", "trece", "catorce", "quince", "dieciséis", "diecisiete",
	"dieciocho", "diecinueve", "veinte", "veintiuno", "veintidós", "veintitrés",
	"veinticuatro", "veinticinco", "veintiséis", "veintisiete", "veintiocho", "veintinueve",
}

var tensWord = []string{
	"", "diez", "veinte", "treinta", "cuarenta", "cincuenta",
	"sesenta", "setenta", "ochenta", "noventa",
}

var hundreds = []string{
	"", "ciento", "doscientos", "trescientos", "cuatrocientos",
	"quinientos", "seiscientos", "setecientos", "ochocientos", "novecientos",
}
