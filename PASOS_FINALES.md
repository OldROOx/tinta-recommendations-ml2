# PASOS FINALES — conectar a Railway y terminar

Tu código ya está completo. Esto es lo ÚNICO que falta: conectarlo a la base de
datos real (Railway) y correrlo para que llene la tabla.

> Reemplaza en todos lados la URL por la que te pasó Poyo. Ejemplo:
> `postgresql://postgres:CONTRASEÑA@reseau.proxy.rlwy.net:12710/railway`
>
> ⚠️ No subas la URL con contraseña a GitHub. Después de la entrega, pídele a
> Poyo que regenere las credenciales en Railway.

---

## Paso 1 — Verifica en qué schema están las tablas

```bash
psql "TU_DATABASE_URL" \
  -c "SELECT table_schema, table_name FROM information_schema.tables WHERE table_name IN ('recommendations','users') ORDER BY table_schema;"
```

- Si dice **public** → no agregues nada a la URL (lo más probable en Railway).
- Si dice **recommendations** → agrega `?search_path=recommendations` al final de la URL.

## Paso 2 — Crea tu tabla recommended_books

```bash
psql "TU_DATABASE_URL" \
  -f migrations/001_create_mvp_reco_tables.up.sql
```

## Paso 3 — Saca un user_id real

```bash
psql "TU_DATABASE_URL" -c "SELECT id, email FROM users LIMIT 5;"
```

Copia un `id` (UUID). Si la tabla `users` está en otro schema, usa `identity.users`.
Si está vacía, para la demo puedes usar cualquier UUID válido.

## Paso 4 — Corre tu servicio

Opción segura (recomendada): pon la URL en un archivo `.env` (no se sube a Git):

```bash
cp .env.example .env
# edita .env y pega tu DATABASE_URL real
go mod tidy
go run . <EL-UUID-DEL-PASO-3> ejemplo_libro.txt "cuantos huesos tiene el craneo"
```

O sin `.env`, exportando la variable a mano:

```bash
go mod tidy
export DATABASE_URL="TU_DATABASE_URL"
go run . <EL-UUID-DEL-PASO-3> ejemplo_libro.txt "cuantos huesos tiene el craneo"
```

Salida esperada: `listo: 5 recomendaciones guardadas para <uuid>`

## Paso 5 — Verifica

```bash
psql "TU_DATABASE_URL" -c "SELECT score, cluster_id, source FROM recommendations;"
psql "TU_DATABASE_URL" -c "SELECT title FROM recommended_books;"
```

Si ves filas (y en Railway la tabla ya no dice "This table is empty"), TERMINASTE. 🎯

---

## Si algo falla

- **Error de SSL** → agrega `?sslmode=require` al final de la URL.
- **Error de FK en user_id** → usa un `id` real del paso 3 (no inventado).
- **"connection refused" / timeout** → revisa que la URL sea la PÚBLICA (proxy.rlwy.net), no la interna.
- **"relation recommendations does not exist"** → la tabla no está en esa base; confírmalo con Poyo.

## Las veces siguientes (atajo)

Una vez hecho el paso 2 (la tabla ya existe), solo repites:

```bash
export DATABASE_URL="TU_DATABASE_URL"
go run . <TU-UUID> ejemplo_libro.txt "tu pregunta"
```
