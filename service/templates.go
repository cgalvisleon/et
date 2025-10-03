package service

const (
	templateOTPEmail = `
	<html>
	<head></head>
	<body>
		<p>Hola, tu código de verificación es <b>{{code}}</b>.</p>
		<p>Recuerda que este código tiene una vigencia de 5 minutos.</p>
	</body>
	</html>`

	templateOTPSMS = "Hola, tu código de verificación es {{code}}.\nRecuerda que este código tiene una vigencia de 5 minutos."
)
