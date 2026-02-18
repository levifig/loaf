# Rails Security

## Rails 8 Authentication

`bin/rails generate authentication` — creates User with `has_secure_password`, Session model, controllers.

## Security Checklist

- [ ] `has_secure_password` for user authentication
- [ ] Strong parameters for all user input (never permit `:role`, `:admin` in public signup)
- [ ] Parameterized SQL queries only (never string interpolation)
- [ ] Credentials in `credentials.yml.enc`
- [ ] Force SSL in production
- [ ] CSRF protection enabled (skip only for token-authenticated APIs)
- [ ] Content Security Policy configured
- [ ] Sessions: `httponly: true`, `same_site: :lax`, `secure: true` in production
