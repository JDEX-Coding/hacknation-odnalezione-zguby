# System "Odnalezione Zguby" - Integracja z dane.gov.pl

Projekt systemu realizujący wyzwanie hackathonowe, ułatwiający samorządom szybkie (max. 5 kroków) i ustandaryzowane wgrywanie danych o rzeczach znalezionych do portalu **dane.gov.pl**. System wykorzystuje AI do opisywania zdjęć oraz wektoryzację (Qdrant) dla wyszukiwania semantycznego, spełniając wymóg dostarczania danych w formacie czytelnym maszynowo.

## ⚙️ Architektura Mikroserwisów

System zbudowany z 4 niezależnych serwisów + infrastruktura wspólna:

### 🖥️ Serwisy Biznesowe

1. **Service A: Gateway (Go + HTMX)**

    - **Port:** 8080
    - **Rola:** Frontend dla urzędników + punkt wejścia danych
    - **Odpowiedzialność:**
        - UI formularza HTMX dla raportowania rzeczy znalezionych
        - Integracja z Vision API (GPT-4o/LLaVA) do real-time analizy zdjęć
        - Upload zdjęć do MinIO
        - Walidacja i publikacja zdarzenia `item.submitted` do RabbitMQ

2. **Service B: CLIP Worker (Python)**

    - **Rola:** Przetwarzanie AI i wektoryzacja
    - **Odpowiedzialność:**
        - Konsumpcja zdarzeń z kolejki `q.lost-items.ingest`
        - Generowanie embeddingów (384-dim) przy użyciu CLIP
        - Zapis wektorów do bazy Qdrant
        - Publikacja zdarzenia `item.vectorized` do RabbitMQ
    - **Status:** _W planie_

3. **Service C: Publisher (Go)**

    - **Rola:** Integracja z dane.gov.pl
    - **Odpowiedzialność:**
        - Konsumpcja przetworzonych danych z kolejki `q.lost-items.publish`
        - Konwersja danych do standardu DCAT-AP PL (JSON-LD/CSV)
        - Autoryzacja i wysyłka danych do API dane.gov.pl
    - **Status:** _W planie_

4. **Service D: Qdrant Vector DB (Go)**
    - **Port:** 6333 (HTTP only, exposed for console)
    - **gRPC:** 6334 (internal only)
    - **Rola:** Baza danych wektorowych
    - **Odpowiedzialność:**
        - Przechowywanie embeddingów przedmiotów
        - Wyszukiwanie semantyczne (cosine similarity)
        - Zarządzanie kolekcjami i metadanymi

### 🏗️ Infrastruktura Wspólna

| Serwis   | Port(s)     | Rola                                     |
| -------- | ----------- | ---------------------------------------- |
| RabbitMQ | 5672, 15672 | Message Broker (Topic Exchange + Queues) |
| MinIO    | 9000, 9001  | S3-compatible object storage (zdjęcia)   |
| Qdrant   | 6333        | Vector database dla semantic search      |

---

## 📊 Diagram Przepływu Danych

```mermaid
graph TB
    %% Aktorzy i Systemy Zewnetrzne
    User((👤 Urzędnik))
    VisionAPI["🤖 Vision API<br/>(GPT-4o/LLaVA)"]
    DaneGov["🏛️ API dane.gov.pl"]

    %% MinIO Storage
    MinIO["📦 MinIO<br/>(S3 Storage)"]

    %% Definicja RabbitMQ
    subgraph MQ["🐇 RabbitMQ Broker<br/>Exchange: lost-found.events"]
        Exchange((Topic Exchange))
        Q_Ingest["📥 Queue<br/>q.lost-items.ingest<br/>routing_key: item.submitted"]
        Q_Publish["📤 Queue<br/>q.lost-items.publish<br/>routing_key: item.vectorized"]
    end

    %% Qdrant Vector DB
    subgraph VectorDB["📊 Qdrant Vector DB<br/>Port: 6333 (HTTP only)<br/>gRPC: 6334 (internal)"]
        QdrantColl["Collection: lost_items<br/>(384-dim vectors)"]
    end

    %% Serwis A: Gateway
    subgraph SvcA["🖥️ Service A: Gateway<br/>Port: 8080<br/>Tech: Go + HTMX"]
        FormUI["Form Handler"]
        MinIOUpload["MinIO Uploader"]
        VisionClient["Vision API Client"]
        Publisher["Event Publisher"]
    end

    %% Serwis B: CLIP Worker
    subgraph SvcB["🐍 Service B: CLIP Worker<br/>Tech: Python<br/>Status: W planie"]
        Consumer["RabbitMQ Consumer"]
        ClipEngine["CLIP Embedding<br/>Engine"]
        QdrantWriter["Qdrant Upsert"]
        Publisher2["Event Publisher"]
    end

    %% Serwis C: Publisher
    subgraph SvcC["📤 Service C: Publisher<br/>Tech: Go<br/>Status: W planie"]
        Consumer2["RabbitMQ Consumer"]
        DataFormatter["DCAT-AP PL<br/>Formatter"]
        GovPublisher["dane.gov.pl<br/>API Client"]
    end

    %% Styles
    style MQ fill:#ff9900,stroke:#333,color:#fff,stroke-width:3px
    style SvcA fill:#00ADD8,stroke:#333,color:#fff,stroke-width:2px
    style SvcB fill:#3776AB,stroke:#333,color:#fff,stroke-width:2px
    style SvcC fill:#00ADD8,stroke:#333,color:#fff,stroke-width:2px
    style VectorDB fill:#0071C5,stroke:#333,color:#fff,stroke-width:2px
    style MinIO fill:#C72C48,stroke:#333,color:#fff,stroke-width:2px

    %% --- PRZEPŁYW DANYCH ---

    %% 1️⃣ Interakcja Urzędnika z Gateway
    User -->|1. Otwiera formularz| FormUI
    User -->|2. Wgrywa zdjęcie| FormUI
    FormUI -->|3. Pyta o opis| VisionClient
    VisionClient -->|4. Wysyła obraz| VisionAPI
    VisionAPI -->|5. Zwraca opis + metadane| VisionClient
    VisionClient -->|6. Sugeruje pole w formularzu| FormUI

    %% 2️⃣ Upload i publikacja
    FormUI -->|7. Wysyła obraz| MinIOUpload
    MinIOUpload -->|8. Zapisuje| MinIO
    MinIO -->|9. Zwraca URL| MinIOUpload
    FormUI -->|10. Submit z URL zdjęcia| Publisher
    Publisher -->|11. Publikuje item.submitted| Exchange

    %% 3️⃣ Routing w RabbitMQ
    Exchange -->|12. Routing| Q_Ingest

    %% 4️⃣ Przetwarzanie Python CLIP
    Q_Ingest -->|13. Consume| Consumer
    Consumer -->|14. Tekst + metadane| ClipEngine
    ClipEngine -->|15. CLIP embeddings| QdrantWriter
    QdrantWriter -->|16. Upsert wektory| QdrantColl
    QdrantColl -->|17. Zwraca status| QdrantWriter
    QdrantWriter -->|18. Publikuje item.vectorized| Publisher2
    Publisher2 -->|19. Publikuje| Exchange

    %% 5️⃣ Routing do Publisher
    Exchange -->|20. Routing| Q_Publish

    %% 6️⃣ Publikacja do dane.gov.pl
    Q_Publish -->|21. Consume| Consumer2
    Consumer2 -->|22. Dane + wektor ID| DataFormatter
    DataFormatter -->|23. DCAT-AP PL JSON| GovPublisher
    GovPublisher -->|24. POST do API| DaneGov
    DaneGov -->|25. ✅ Confirm| GovPublisher

    classDef success fill:#4CAF50,stroke:#333,color:#fff
    classDef inProgress fill:#FF9800,stroke:#333,color:#fff
```

---

## 🔄 Sekwencja Operacji (Szczegółowo)

1. **Urzędnik wgrywa rzecz znalezioną** → Service A (Gateway)
2. **Gateway analizuje zdjęcie** → Vision API (real-time)
3. **Gateway zapisuje obraz** → MinIO (S3 storage)
4. **Gateway publikuje event** → RabbitMQ (item.submitted)
5. **Python CLIP Worker konsumuje** → Generuje embeddings
6. **Embeddings zapisane** → Qdrant Vector DB
7. **Publikuje zdarzenie** → RabbitMQ (item.vectorized)
8. **Go Publisher konsumuje** → Konwertuje do standardu
9. **Publikuje do dane.gov.pl** → Integracja rządowa ✅

---

Wszystkie wymagane serwisy są teraz uruchomione:

-   🐇 **RabbitMQ**: http://localhost:15672 (admin/admin123)
-   📦 **MinIO**: http://localhost:9001 (minioadmin/minioadmin123)
-   🔍 **Qdrant**: http://localhost:6333/dashboard

## 🐳 Docker Compose - Serwisy

Projekt używa Docker Compose do orkiestracji wszystkich serwisów. Plik `docker-compose.yml` zawiera konfigurację dla:

### Serwisy Aplikacyjne

| Serwis             | Container      | Port   | Dockerfile         |
| ------------------ | -------------- | ------ | ------------------ |
| **Gateway**        | a-gateway      | 8080   | service-a-gateway/ |
| **Qdrant Service** | qdrant-service | internal\* | qdrant-service/    |

### Infrastruktura

| Serwis        | Image                    | Ports       | Rola                  |
| ------------- | ------------------------ | ----------- | --------------------- |
| **RabbitMQ**  | rabbitmq:3.12-management | 5672, 15672 | Message Broker        |
| **Qdrant DB** | qdrant/qdrant:latest     | 6333        | Vector Database       |
| **MinIO**     | minio/minio:latest       | 9000, 9001  | S3-compatible Storage |

### Wolumeny

-   `rabbitmq_data` - Dane RabbitMQ
-   `qdrant_data` - Dane Qdrant
-   `minio_data` - Dane MinIO
-   `./qdrant_storage` - Host storage dla Qdrant (opcjonalne)

### Sieci

-   `odnalezione-network` - Sieć bridge łącząca wszystkie serwisy

---

## 📋 Wymagane Zmienne Środowiska

### Service A: Gateway

```env
GATEWAY_PORT=8080
RABBITMQ_URL=amqp://admin:admin123@rabbitmq:5672/
MINIO_ENDPOINT=minio:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin123
MINIO_USE_SSL=false
VISION_API_KEY=<your-api-key>  # np. OpenAI API key
```

### Service B: CLIP Worker (Python) - _W planie_

```env
RABBITMQ_URL=amqp://admin:admin123@rabbitmq:5672/
QDRANT_URL=http://qdrant-service:8080
COLLECTION_NAME=lost_items
```

### Service C: Publisher (Go) - _W planie_

```env
RABBITMQ_URL=amqp://admin:admin123@rabbitmq:5672/
DANE_GOV_API_URL=https://api.dane.gov.pl/...
DANE_GOV_API_KEY=<your-api-key>
```

---

## 🔌 RabbitMQ Configuration

System jest wstępnie skonfigurowany z:

-   **Exchange**: `lost-found.events` (topic)
-   **Queue 1**: `q.lost-items.ingest` (routing key: `item.submitted`)
-   **Queue 2**: `q.lost-items.publish` (routing key: `item.vectorized`)

Konfiguracja wykonywana automatycznie przez `rabbitmq-init.sh` podczas uruchamiania.
