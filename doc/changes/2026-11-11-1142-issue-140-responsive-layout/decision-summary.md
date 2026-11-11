### Resumen de decisiones de revisión actualizado (#140, informativo)

| Hallazgo | Criticidad | Decisión | Acción |
|---|---|---|---|
| REV-001 Tablas cortadas entre 672 y 1055 px (QA-F001) | crítico | fix | Tablas apiladas hasta `max-width: 1055.98px`. Caso de uso UC-TI-05 (`doc/use-cases/tracked-items/UC-TI-05-review-tracked-items-at-tablet-width.md`). Test de regresión "QA-F001 …" en `tests/responsive-layout.spec.ts` (700/800/1000 px). Según el desarrollador, falla con el breakpoint anterior; el revisor no repitió esa comprobación. Con la corrección pasa (23/23 en la ejecución del revisor). Corrección confirmada. |
| REV-002 La tabla apilada puede perder semántica en lectores de pantalla | no crítico | fix (solo verificación) | Cerrado: QA-014 pasa, cada valor se lee con su etiqueta (solo Chromium). |
| REV-003 Reglas de ajuste de texto también en escritorio | no crítico | reject | Cerrado: QA-015 sin diferencias frente a master a 1056/1440 px. |
| REV-004 Selectores CSS no listados en la especificación técnica | no crítico | fix (docs) | Documentados en technical.md. |
| REV-005 Los tests de admin a anchura de tablet solo comprueban el scroll de página | no crítico | defer | El CSS compartido está cubierto por el test QA-F001; queda como tarea pendiente (doc/todo). |
