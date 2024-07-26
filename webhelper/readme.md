Very simple wrapper classes to help use http.HandleFunc & cie.
The 'handler' funcs receive a Context and return a go error.
Handlers can call Render to render templates from go's html/template.
This uses go 1.22> http.HandleFunc("GET "+pattern, http.HandleFunc)