package main

import (
	"context"
	"html/template"
	"log"
	"net/http"

	"github.com/google/uuid"
)

var (
	homeTemplate = template.Must(template.ParseFiles("templates/home.html"))
)

func refreshCookies(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, c := range r.Cookies() {
			c.MaxAge = cookieMaxAge
			http.SetCookie(w, c)
		}
		next(w, r)
	}
}

func ensureSessionID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var sessionID string
		c, err := r.Cookie(cookieSessionID)
		if err == http.ErrNoCookie {
			u, _ := uuid.NewRandom()
			sessionID = u.String()
			http.SetCookie(w, &http.Cookie{
				Name:   cookieSessionID,
				Value:  sessionID,
				MaxAge: cookieMaxAge,
			})
		} else if err != nil {
			log.Printf("unrecognized cookie error: %+v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else {
			sessionID = c.Value
		}
		ctx := context.WithValue(r.Context(), ctxKeySessionID{}, sessionID)
		r = r.WithContext(ctx)
		next(w, r)
	}
}

func (fe *frontendServer) homeHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[home] session_id=%+v", r.Context().Value(ctxKeySessionID{}))
	currencies, err := fe.getCurrencies(r.Context())
	if err != nil {
		log.Println(err) // TODO(ahmetb) use structured logging
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf("currencies: %+v", currencies)
	products, err := fe.getProducts(r.Context())
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf("# products: %d", len(products))

	if err := homeTemplate.Execute(w, map[string]interface{}{
		"user_currency": currentCurrency(r),
		"currencies":    currencies,
		"products":      products,
	}); err != nil {
		log.Println(err)
	}
}

func (fe *frontendServer) logoutHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[home] session_id=%+v", r.Context().Value(ctxKeySessionID{}))

	// clear all cookies
	for _, c := range r.Cookies() {
		c.MaxAge = -1
		http.SetCookie(w, c)
	}
	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusFound)
}

func (fe *frontendServer) setCurrencyHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[setCurrency] session_id=%+v", r.Context().Value(ctxKeySessionID{}))
	cur := r.FormValue("currency_code")
	if cur != "" {
		http.SetCookie(w, &http.Cookie{
			Name:   cookieCurrency,
			Value:  cur,
			MaxAge: cookieMaxAge,
		})
	}
	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusFound)
}

func currentCurrency(r *http.Request) string {
	c, _ := r.Cookie(cookieCurrency)
	if c != nil {
		return c.Value
	}
	return defaultCurrency
}