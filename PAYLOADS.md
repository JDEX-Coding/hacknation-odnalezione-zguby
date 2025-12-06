# 📨 Event Payloads - Struktura Danych

Dokumentacja struktur JSON dla wszystkich zdarzeń przesyłanych przez RabbitMQ w systemie "Odnalezione Zguby".

---

## 📥 Event #1: item.submitted

**Wysyłany przez:** Service A (Gateway)
**Konsumowany przez:** Service B (CLIP Worker)
**Routing Key:** `item.submitted`
**Queue:** `q.lost-items.ingest`

### Schema

```json
{
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "Znaleziony portfel",
    "description": "Czarny portfel skórzany ze znalezionym dowodem osobistym",
    "category": "Portfele i torby",
    "location": "Warszawa, Rynek Starego Miasta",
    "found_date": "2024-12-06T10:30:00Z",
    "image_url": "http://minio:9000/lost-items-images/uploads/2024-12-06/550e8400-e29b-41d4-a716-446655440000.jpg",
    "contact_info": "biuro@urzad.pl",
    "timestamp": "2024-12-06T10:35:00Z"
}
```

### Pole Description

| Pole           | Typ      | Wymagane | Opis                                                              |
| -------------- | -------- | -------- | ----------------------------------------------------------------- |
| `id`           | UUID     | ✅       | Unikalny identyfikator przedmiotu                                 |
| `title`        | string   | ✅       | Krótki tytuł przedmiotu                                           |
| `description`  | string   | ✅       | Pełny opis przedmiotu (mogą być to sugestie z Vision API)         |
| `category`     | string   | ✅       | Kategoria przedmiotu (np. "Elektronika", "Portfele", "Klucze")    |
| `location`     | string   | ✅       | Miejsce znalezienia (gmina/dzielnica, bez dokładnych koordynatów) |
| `found_date`   | ISO 8601 | ✅       | Data znalezienia przedmiotu                                       |
| `image_url`    | URL      | ✅       | Pełny adres URL zdjęcia w MinIO                                   |
| `contact_info` | string   | ✅       | Dane kontaktowe (email/telefon urzędu)                            |
| `timestamp`    | ISO 8601 | ✅       | Timestamp publikacji zdarzenia                                    |

---

## 📤 Event #2: item.vectorized

**Wysyłany przez:** Service B (CLIP Worker)
**Konsumowany przez:** Service C (Publisher)
**Routing Key:** `item.vectorized`
**Queue:** `q.lost-items.publish`

### Schema

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "original_data": {
    "title": "Znaleziony portfel",
    "description": "Czarny portfel skórzany ze znalezionym dowodem osobistym",
    "category": "Portfele i torby",
    "location": "Warszawa, Rynek Starego Miasta",
    "image_url": "http://minio:9000/lost-items-images/uploads/2024-12-06/550e8400-e29b-41d4-a716-446655440000.jpg",
    "contact_info": "biuro@urzad.pl"
  },
  "vector_embedding": [0.123, 0.456, -0.789, ..., 0.321],
  "vector_id": "qdrant-vector-id-12345",
  "embedding_model": "CLIP",
  "embedding_dimension": 384,
  "processing_time_ms": 2345,
  "processed_at": "2024-12-06T10:37:00Z"
}
```

### Pole Description

| Pole                  | Typ       | Wymagane | Opis                                                   |
| --------------------- | --------- | -------- | ------------------------------------------------------ |
| `id`                  | UUID      | ✅       | Unikalny identyfikator (taki sam jak w item.submitted) |
| `request_id`          | UUID      | ✅       | ID oryginalnego żądania (tracking)                     |
| `original_data`       | object    | ✅       | Kopia danych z item.submitted                          |
| `vector_embedding`    | float32[] | ✅       | Tablica 384-wymiarowych wektorów CLIP                  |
| `vector_id`           | string    | ✅       | ID wektora w bazie Qdrant                              |
| `embedding_model`     | string    | ✅       | Model użyty do generacji ("CLIP")                      |
| `embedding_dimension` | integer   | ✅       | Wymiar wektora (384)                                   |
| `processing_time_ms`  | integer   | ✅       | Czas przetworzenia w ms                                |
| `processed_at`        | ISO 8601  | ✅       | Timestamp przetworzenia                                |

---

## 📤 dane.gov.pl Export Format

**Format do wysłania:** Zstandaryzowany JSON-LD (DCAT-AP PL)

### DCAT-AP PL Schema

```json
{
    "id_ewidencyjny": "550e8400-e29b-41d4-a716-446655440000",
    "nazwa_przedmiotu": "Portfel czarny skórzany",
    "kategoria": "Portfele i torby",
    "data_znalezienia": "2024-12-06",
    "miejsce_gmina": "Warszawa",
    "miejsce_opis": "Rynek Starego Miasta (bez dokładnych koordynatów)",
    "cechy_szczegolne": "Czarny portfel skórzany ze znalezionym dowodem osobistym",
    "jednostka_zglaszajaca": "Odnalezione Zguby v1",
    "link_do_zdjecia": "http://minio:9000/lost-items-images/uploads/2024-12-06/550e8400-e29b-41d4-a716-446655440000.jpg",
    "status": "Do odbioru",
    "data_publikacji": "2024-12-06T10:37:00Z"
}
```

### Pole Description

| Pole                    | Opis                                                      |
| ----------------------- | --------------------------------------------------------- |
| `id_ewidencyjny`        | Nasz wewnętrzny UUID                                      |
| `nazwa_przedmiotu`      | Nazwa przedmiotu (ze słowami kluczowymi)                  |
| `kategoria`             | Kategoria zgodna ze słownikiem dane.gov.pl                |
| `data_znalezienia`      | YYYY-MM-DD format                                         |
| `miejsce_gmina`         | Gmina/Miasto                                              |
| `miejsce_opis`          | Opis lokalizacji BEZ dokładnych koordynatów (RODO)        |
| `cechy_szczegolne`      | Cechy identyfikujące przedmiot                            |
| `jednostka_zglaszajaca` | Nazwa systemu/projektu                                    |
| `link_do_zdjecia`       | Publiczny URL do zdjęcia                                  |
| `status`                | Status przedmiotu (Do odbioru, Odebrany, Zagubiony, itp.) |
| `data_publikacji`       | Kiedy dana została opublikowana                           |

---

## 🏠 Wrapper API dane.gov.pl

W razie konieczności wysłania przez REST API:

```json
{
    "data": {
        "type": "resource",
        "attributes": {
            "title": "Znalezione przedmioty",
            "description": "Rejestr rzeczy znalezionych w gminie Warszawa",
            "resources": [
                {
                    "url": "http://api.example.com/lost-items.json",
                    "format": "JSON-LD",
                    "description": "Dane w formacie JSON-LD"
                }
            ]
        }
    }
}
```

---

## 🧪 Przykład Całego Przepływu

### 1️⃣ Urzędnik wysyła formularz

```
POST /create
Content-Type: multipart/form-data

title: "Portfel"
description: "Czarny portfel ze złotą klamrą"
category: "Portfele i torby"
location: "Rynek Starego Miasta"
found_date: "2024-12-06"
image: <binary>
contact_info: "biuro@urzad.pl"
```

### 2️⃣ Gateway publikuje item.submitted

```json
{
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "Portfel",
    "description": "Czarny portfel ze złotą klamrą",
    "category": "Portfele i torby",
    "location": "Rynek Starego Miasta",
    "found_date": "2024-12-06T00:00:00Z",
    "image_url": "http://minio:9000/lost-items-images/uploads/2024-12-06/550e8400-e29b-41d4-a716-446655440000.jpg",
    "contact_info": "biuro@urzad.pl",
    "timestamp": "2024-12-06T10:35:00Z"
}
```

### 3️⃣ CLIP Worker generuje embedding i publikuje item.vectorized

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "original_data": { /* jak wyżej */ },
  "vector_embedding": [0.123, 0.456, -0.789, ...],
  "vector_id": "qdrant-vector-id-12345",
  "embedding_model": "CLIP",
  "embedding_dimension": 384,
  "processing_time_ms": 2345,
  "processed_at": "2024-12-06T10:37:00Z"
}
```

### 4️⃣ Publisher wysyła do dane.gov.pl

```json
POST https://api.dane.gov.pl/resources
Authorization: Bearer TOKEN
Content-Type: application/json

{
  "id_ewidencyjny": "550e8400-e29b-41d4-a716-446655440000",
  "nazwa_przedmiotu": "Portfel czarny skórzany",
  "kategoria": "Portfele i torby",
  "data_znalezienia": "2024-12-06",
  "miejsce_gmina": "Warszawa",
  "miejsce_opis": "Rynek Starego Miasta",
  "cechy_szczegolne": "Czarny portfel ze złotą klamrą",
  "jednostka_zglaszajaca": "Odnalezione Zguby v1",
  "link_do_zdjecia": "http://minio:9000/lost-items-images/uploads/2024-12-06/550e8400-e29b-41d4-a716-446655440000.jpg",
  "status": "Do odbioru",
  "data_publikacji": "2024-12-06T10:37:00Z"
}
```

---

## 🔒 Bezpieczeństwo i RODO

⚠️ **WAŻNE: Wszystkie payloady muszą być przefiltrowane**

-   ✅ **DOZWOLONE:** Nazwa przedmiotu, kategoria, gmina, ogólny opis cechy
-   ❌ **ZABRONIONE:** Dokładne koordynaty GPS, imiona/nazwiska znalazcy, numery telefonów prywatnych
-   ❌ **MASKOWANE:** Adresy e-mail → adresy urzędów
