package ia

import "testing"

func TestIsListFactsIntent(t *testing.T) {
	positives := []string{
		"que hechos tienes en la base de conocimiento",
		"que sabes",
		"cuentame que sabes",
		"resumen de lo que sabes",
		"que informacion tienes",
	}
	for _, q := range positives {
		if !isListFactsIntent(q) {
			t.Errorf("expected %q to be detected as a list-facts question", q)
		}
	}

	negatives := []string{
		"donde estuve el lunes 3 de marzo",
		"compre el carro en la agencia el 12 de abril",
		"es verdad que llovio ayer",
	}
	for _, q := range negatives {
		if isListFactsIntent(q) {
			t.Errorf("expected %q to NOT be detected as a list-facts question", q)
		}
	}
}

func TestIsGreetingIntent(t *testing.T) {
	positives := []string{"hola", "Hola!", "buenos dias", "gracias", "como estas"}
	for _, q := range positives {
		if !isGreetingIntent(q) {
			t.Errorf("expected %q to be detected as a greeting", q)
		}
	}

	negatives := []string{
		"el cielo tiene color",
		"donde estuve el lunes",
		"que hechos tienes en la base de conocimiento",
	}
	for _, q := range negatives {
		if isGreetingIntent(q) {
			t.Errorf("expected %q to NOT be detected as a greeting", q)
		}
	}
}
