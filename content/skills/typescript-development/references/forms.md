# Form Patterns

## Stack

| Component | Tool |
|-----------|------|
| Form Library | React Hook Form |
| Validation | Zod |
| Submission | Server Actions |

## Conventions

- Define Zod schema first, derive form type with `z.infer<typeof schema>`
- Use `zodResolver` to connect schema to React Hook Form
- Validate on both client (Zod) and server (Server Actions with `safeParse`)
- Server Actions return `{ errors?: Record<string, string[]>; success?: boolean }`
- Use `useFormState` + `useFormStatus` for server action forms

## Zod Patterns

- Cross-field validation: `.refine()` with `path` for field-level errors
- Discriminated unions: `z.discriminatedUnion("type", [...])`
- File validation: `z.instanceof(File).refine(f => f.size <= 5_000_000)`
- Transform: `z.string().transform(str => new Date(str))`

## Critical Rules

- Always validate on both client and server
- Include proper ARIA attributes (`aria-invalid`, `aria-describedby`, `role="alert"`)
- Handle submission states (loading, disabled)
- Use uncontrolled inputs by default (React Hook Form's `register`)
