# Resumen de decisiones de revisión (issue #140, lista a todo el ancho en móvil)

| Hallazgo | Criticidad | Decisión | Acción |
|---|---|---|---|
| REV-001: el margen negativo no descontaba el padding de 1rem del `TabPanel` de Carbon (`.cds--tab-content`); las tarjetas seguían con 16 px de margen lateral | Crítica (la funcionalidad pedida no funcionaba) | Corregir | Corregido: el `calc` incluye el padding del panel (64 px por lado); revisor lo verificó leyendo el código; caso de uso UC-TI-06 añadido y enlazado; tests Playwright de 320/360 px escritos. **No confirmado: pendiente de ejecución** (requiere que pasen en CI) |
| REV-002: el test de 700 px comparaba con `.page-shell` en lugar de la caja del panel de pestañas | No crítica (solo afecta a la verificación) | Corregir | Corregido: compara con la fila de búsqueda `.tracked-table-search`; verificado por el revisor; no ejecutado |

Limitación: Chromium no arranca en este entorno (falta `libnspr4.so`, sin sudo) y el QA en navegador quedó bloqueado. No se debe fusionar hasta que `tests/responsive-layout.spec.ts` pase en CI; si falla, REV-001 se reabre como crítico.
