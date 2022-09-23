package bot

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"RacoBot/internal/db"
	"RacoBot/pkg/fibapi"
)

func TestNoticeMessage_String(t *testing.T) {
	type test struct {
		raw            string
		subjectUPCCode int64
		userLangCode   string
		want           string
	}
	tests := []test{
		{
			`{"id": 123521,"titol": "Inicio del curso","codi_assig": "SI","text": "<p>Hola a todos,</p>\r\n<p>bienvenido a este curso de SI.</p>\r\n<p>Como ya sabéis, las clases de teoria empezarán este lunes. Las clases de laboratorio empezarán en marzo, publicaremos el calendario en el Racó y en Atenea próximamente.</p>\r\n<p>Usaremos principalmente Atenea para la publicación de todo el material, las presentaciones de teoría, los enunciados, los cuestionarios y las entregas de laboratorio y los controles y exámenes de los cursos anteriores.</p>\r\n<p>Usaremos en cambio el Racó para la publicación de los avisos.</p>\r\n<p>Saludos,<br />Davide </p>","data_insercio": "2022-02-12T00:00:00","data_modificacio": "2022-02-12T10:56:41","data_caducitat": "2022-07-20T00:00:00","adjunts": []}`,
			270123,
			"en",
			"[#SI] <b>Inicio del curso</b>\n\n<i>12/02/2022 10:56:41</i>  <a href=\"https://raco.fib.upc.edu/avisos/veure.jsp?espai=270123&id=123521\">Link</a>\n\nHola a todos,\r\nbienvenido a este curso de SI.\r\nComo ya sabéis, las clases de teoria empezarán este lunes. Las clases de laboratorio empezarán en marzo, publicaremos el calendario en el Racó y en Atenea próximamente.\r\nUsaremos principalmente Atenea para la publicación de todo el material, las presentaciones de teoría, los enunciados, los cuestionarios y las entregas de laboratorio y los controles y exámenes de los cursos anteriores.\r\nUsaremos en cambio el Racó para la publicación de los avisos.\r\nSaludos,\nDavide ",
		},
		{
			`{"id": 123522,"titol": "Inicio del curso","codi_assig": "PROP","text": "<p>Bienvenidos a PROP. Varias informaciones de interés de cara al comienzo del curso:</p>\r\n<p>- Adjunto un calendario &#34;aproximado&#34; de las sesiones de teoría</p>\r\n<p>- Los laboratorios de la primera semana de clase <strong></strong>se dedicarán a resolver un caso práctico. De manera excepcional, esta semana no habrá clases en el <strong>grupo 12</strong>. Así pues, los estudiantes de ese grupo pueden asistir a cualquiera de las 5 clases de laboratorio de los otros grupos, donde se explicará el mismo contenido.</p>\r\n<p>- La segunda clase de laboratorio se dedicará, entre otras cosas, a formar los equipos para el proyecto. Es MUY IMPORTANTE asistir a esa segunda sesión.</p>\r\n<p>- Es MUY CONVENIENTE haberse leído el documento &#34;Normativa i descripcions dels lliuraments&#34; que está en la web de la asignatura (y que adjunto)</p>","data_insercio": "2022-02-12T00:00:00","data_modificacio": "2022-02-12T11:29:37","data_caducitat": "2022-07-20T00:00:00","adjunts": [    {"tipus_mime": "application/pdf","nom": "Calendario_Sesiones_Teoria_PROP_-2q2122.pdf","url": "https://api.fib.upc.edu/v2/jo/avisos/adjunt/96611","data_modificacio": "2022-02-12T04:24:35","mida": 66670},{"tipus_mime": "application/pdf","nom": "Normativa-2q2122.pdf","url": "https://api.fib.upc.edu/v2/jo/avisos/adjunt/96612","data_modificacio": "2022-02-12T04:24:35","mida": 121304}]}`,
			270017,
			"es",
			"[#PROP] <b>Inicio del curso</b>\n\n<i>12/02/2022 11:29:37</i>  <a href=\"https://raco.fib.upc.edu/avisos/veure.jsp?espai=270017&id=123522\">Enlace</a>\n\nBienvenidos a PROP. Varias informaciones de interés de cara al comienzo del curso:\r\n- Adjunto un calendario \"aproximado\" de las sesiones de teoría\r\n- Los laboratorios de la primera semana de clase <strong></strong>se dedicarán a resolver un caso práctico. De manera excepcional, esta semana no habrá clases en el <strong>grupo 12</strong>. Así pues, los estudiantes de ese grupo pueden asistir a cualquiera de las 5 clases de laboratorio de los otros grupos, donde se explicará el mismo contenido.\r\n- La segunda clase de laboratorio se dedicará, entre otras cosas, a formar los equipos para el proyecto. Es MUY IMPORTANTE asistir a esa segunda sesión.\r\n- Es MUY CONVENIENTE haberse leído el documento \"Normativa i descripcions dels lliuraments\" que está en la web de la asignatura (y que adjunto)\n\n<i>📎 Con 2 adjuntos:</i>\n<a href=\"https://api.fib.upc.edu/v2/accounts/login/?next=https%3A%2F%2Fapi.fib.upc.edu%2Fv2%2Fjo%2Favisos%2Fadjunt%2F96611\">Calendario_Sesiones_Teoria_PROP_-2q2122.pdf</a>  (65,1 KiB)\n<a href=\"https://api.fib.upc.edu/v2/accounts/login/?next=https%3A%2F%2Fapi.fib.upc.edu%2Fv2%2Fjo%2Favisos%2Fadjunt%2F96612\">Normativa-2q2122.pdf</a>  (118,5 KiB)",
		},
		{
			`{"id": 126594,"titol": "Prematrícula d'assignatures d'especialitat","codi_assig": "#PREMAT-GEI","text": "<p>Si et queden assignatures obligatories d'especialitat o b&eacute; aquest proper quadrimestre has de triar l'especialitat, no oblidis que per assegurar pla&ccedil;a en un grup concret haur&agrave;s de fer la prematr&iacute;cula al Rac&oacute;.</p>\r\n<p>L'aplicaci&oacute; de prematr&iacute;cula estar&agrave; disponible des de dilluns dia 11 a les 10:00 fins dimarts dia 12 a mitjanit. En funci&oacute; dels grups triats, s'intentar&agrave; obrir suficients places perque ning&uacute; es quedi sense lloc. Dijous 14 es podran fer modificacions</p>\r\n<p><a href=\"https://www.fib.upc.edu/ca/estudis/secretaria/tramits/prematricula-de-les-assignatures-despecialitat-del-gei\">https://www.fib.upc.edu/ca/estudis/secretaria/tramits/prematricula-de-les-assignatures-despecialitat-del-gei</a></p>\r\n<ul>\r\n<li><a href=\"https://raco.fib.upc.edu/servlet/raco.prematricula.CarregaAssignaturesPrematricula\">Accedir a l'aplicaci&oacute; de prematricula</a></li>\r\n</ul>\r\n<p><a href=\"https://www.fib.upc.edu/ca/estudis/secretaria/tramits/prematricula-de-les-assignatures-despecialitat-del-gei\"></a></p>","data_insercio": "2022-07-05T09:25:50","data_modificacio": "2022-07-05T00:00:00","data_caducitat": "2022-07-15T00:00:00","adjunts": []}`,
			0,
			"ca",
			"[#PREMAT_GEI] <b>Prematrícula d'assignatures d'especialitat</b>\n\n<i>05/07/2022 09:25:50</i>  <a href=\"https://raco.fib.upc.edu/#avis-126594\">Enllaç</a>\n\nSi et queden assignatures obligatories d'especialitat o bé aquest proper quadrimestre has de triar l'especialitat, no oblidis que per assegurar plaça en un grup concret hauràs de fer la prematrícula al Racó.\r\nL'aplicació de prematrícula estarà disponible des de dilluns dia 11 a les 10:00 fins dimarts dia 12 a mitjanit. En funció dels grups triats, s'intentarà obrir suficients places perque ningú es quedi sense lloc. Dijous 14 es podran fer modificacions\r\n<a href=\"https://www.fib.upc.edu/ca/estudis/secretaria/tramits/prematricula-de-les-assignatures-despecialitat-del-gei\">https://www.fib.upc.edu/ca/estudis/secretaria/tramits/prematricula-de-les-assignatures-despecialitat-del-gei</a>\r\n\r\n  • <a href=\"https://raco.fib.upc.edu/servlet/raco.prematricula.CarregaAssignaturesPrematricula\">Accedir a l'aplicació de prematricula</a>\n\r\n\r\n<a href=\"https://www.fib.upc.edu/ca/estudis/secretaria/tramits/prematricula-de-les-assignatures-despecialitat-del-gei\"></a>",
		},
		{
			`{"id":127018,"titol":"Codi Prova","codi_assig":"CI","text":"/* Main.c file generated by New Project wizard * * Created: dg. set. 11 2022 * Processor: PIC18F45K22 * Compiler: MPLAB XC8 */ #include &lt;xc.h&gt; void main(void) { // Write your code here ANSELAbits.ANSA0 &#61; 0; TRISAbits.TRISA0 &#61; 0; while (1) { if (PORTAbits.RA0 &#61;&#61; 1) { PORTAbits.RA0 &#61; 0; } else { PORTAbits.RA0 &#61; 1; } } }","data_insercio":"2022-09-12T00:00:00","data_modificacio":"2022-09-12T09:11:16","data_caducitat":"2023-02-08T00:00:00","adjunts":[]}`,
			270013,
			"en",
			"[#CI] <b>Codi Prova</b>\n\n<i>12/09/2022 09:11:16</i>  <a href=\"https://raco.fib.upc.edu/avisos/veure.jsp?espai=270013&id=127018\">Link</a>\n\n/* Main.c file generated by New Project wizard * * Created: dg. set. 11 2022 * Processor: PIC18F45K22 * Compiler: MPLAB XC8 */ #include &lt;xc.h&gt; void main(void) { // Write your code here ANSELAbits.ANSA0 = 0; TRISAbits.TRISA0 = 0; while (1) { if (PORTAbits.RA0 == 1) { PORTAbits.RA0 = 0; } else { PORTAbits.RA0 = 1; } } }",
		},
	}

	// simulate a too long notice
	var tooLongText string
	for i := 0; i < 4097; i++ {
		tooLongText += "a"
	}
	tests = append(tests, test{
		fmt.Sprintf("{\"id\": 126418,\"titol\": \"Notes finals definitives\",\"codi_assig\": \"AC\",\"text\": \"%s\",\"data_insercio\": \"2022-06-27T08:01:21\",\"data_modificacio\": \"2022-06-27T08:01:21\",\"data_caducitat\": \"2022-08-26T08:01:21\",\"adjunts\": []}", tooLongText),
		270018,
		"ca",
		"[#AC] <b>Notes finals definitives</b>\n\n<i>27/06/2022 08:01:21</i>  <a href=\"https://raco.fib.upc.edu/avisos/veure.jsp?espai=270018&id=126418\">Enllaç</a>\n\nHo sento, però aquest missatge és massa llarg per enviar-lo per Telegram, si us plau consulteu-lo a través <a href=\"https://raco.fib.upc.edu/avisos/veure.jsp?espai=270018&id=126418\">d'aquest enllaç</a>.",
	})

	for _, tt := range tests {
		var notice fibapi.Notice
		if err := json.Unmarshal([]byte(tt.raw), &notice); err != nil {
			t.Error(err)
		}
		t.Run(strconv.FormatInt(int64(notice.ID), 10), func(t *testing.T) {
			var linkURL string
			if strings.HasPrefix(notice.SubjectCode, "#") {
				linkURL = fmt.Sprintf("%s/#avis-%d", racoBaseURL, notice.ID)
			} else {
				linkURL = fmt.Sprintf(racoNoticeURLTemplate, tt.subjectUPCCode, notice.ID)
			}

			m := &NoticeMessage{
				Notice:  notice,
				user:    db.User{LanguageCode: tt.userLangCode},
				linkURL: linkURL,
			}

			if gotResult := m.String(); gotResult != tt.want {
				t.Error(cmp.Diff(tt.want, gotResult))
			}
		})
	}
}
