# System "Odnalezione Zguby" - Integracja z dane.gov.pl

[cite_start]Projekt systemu realizujący wyzwanie hackathonowe, ułatwiający samorządom szybkie (max. 5 kroków) i ustandaryzowane wgrywanie danych o rzeczach znalezionych do portalu **dane.gov.pl**[cite: 12, 20]. [cite_start]System wykorzystuje AI do opisywania zdjęć oraz wektoryzację (Qdrant) dla wyszukiwania semantycznego, spełniając wymóg dostarczania danych w formacie czytelnym maszynowo[cite: 8, 29].

## 🏗 Architektura Systemu

Architektura oparta jest na mikroserwisach i asynchronicznym przetwarzaniu zdarzeń. Składa się z trzech głównych serwisów:

### Komponenty:

1.  **Service A: Gateway (Go + HTMX)**

    -   **Rola:** Interfejs dla urzędnika (Frontend) i punkt wejścia danych.
    -   **Zadania:**
        -   Obsługa formularza HTMX.
        -   Komunikacja z **Vision API** (np. GPT-4o/LLaVA) w czasie rzeczywistym, aby wygenerować opis przedmiotu na podstawie wgranego zdjęcia (wsparcie UX).
        -   Walidacja wstępna i wysłanie zdarzenia `ItemSubmitted` do RabbitMQ.

2.  **Service B: AI Worker (Python)**

    -   **Rola:** Przetwarzanie semantyczne (Heavy lifting).
    -   **Zadania:**
        -   Konsumpcja zdarzeń z kolejki `q.lost-items.ingest`.
        -   Generowanie embeddingów (wektorów) dla tekstu i obrazu.
        -   Zapis metadanych wektorowych do bazy **Qdrant**.
        -   Emisja zdarzenia `ItemVectorized` do RabbitMQ.

3.  **Service C: Publisher (Go)**

    -   **Rola:** Integracja z API rządowym.
    -   **Zadania:**
        -   Konsumpcja przetworzonych danych z kolejki `q.lost-items.publish`.
        -   [cite_start]Konwersja danych do standardu wymaganego przez dane.gov.pl (JSON-LD / CSV)[cite: 29].
        -   Autoryzacja i wysyłka danych (POST) do API portalu.

4.  **Infrastruktura:**
    -   **RabbitMQ:** Message Broker (Topic Exchange `lost-found.events`).
    -   **Qdrant:** Baza wektorowa.

---

### 📊 Diagram Przepływu

```mermaid
graph TD
    %% Aktorzy i Systemy Zewnetrzne
    User((Urzędnik))
    VisionAPI[External Vision API]
    DaneGov[API dane.gov.pl]
    Qdrant[(Qdrant Vector DB)]

    %% Definicja RabbitMQ
    subgraph RabbitMQ_Cluster["RabbitMQ Broker"]
        Exchange((Exchange: lost-found.events))
        Q_Ingest[Queue: q.lost-items.ingest]
        Q_Publish[Queue: q.lost-items.publish]
    end

    %% Serwis A: Gateway
    subgraph Svc_Gateway["Service A: Gateway Go+HTMX"]
        UI_Handler[HTMX Form Handler]
        Img_Helper[Image Helper]
    end

    %% Serwis B: AI Worker
    subgraph Svc_Python["Service B: Python Worker"]
        Py_Consumer[Event Consumer]
        Vector_Logic[Vector Engine]
    end

    %% Serwis C: Publisher
    subgraph Svc_Publisher["Service C: Publisher Go"]
        Go_Consumer[Event Consumer]
        Data_Formatter[Gov Data Formatter]
    end

    %% Styles
    style RabbitMQ_Cluster fill:#ff9900,stroke:#333,color:#fff
    style Svc_Gateway fill:#00ADD8,stroke:#333,color:#fff
    style Svc_Python fill:#3776AB,stroke:#333,color:#fff
    style Svc_Publisher fill:#00ADD8,stroke:#333,color:#fff

    %% --- RELACJE ---

    %% 1. Interakcja Urzędnika
    User -->|1. Formularz + Zdjęcie| UI_Handler
    UI_Handler -.->|2. Get Description| Img_Helper
    Img_Helper -.->|3. Analyze Img| VisionAPI
    VisionAPI -.->|4. Return Text| Img_Helper
    Img_Helper -.->|5. Fill Form| UI_Handler

    %% 2. Wysyłka do Kolejki Ingest
    UI_Handler -->|6. Submit Event| Exchange
    Exchange -->|key: item.submitted| Q_Ingest

    %% 3. Przetwarzanie Python
    Q_Ingest -->|7. Consume| Py_Consumer
    Py_Consumer --> Vector_Logic
    Vector_Logic -->|8. Upsert| Qdrant

    %% 4. Wysyłka do Kolejki Publish
    Vector_Logic -->|9. Publish Event| Exchange
    Exchange -->|key: item.vectorized| Q_Publish

    %% 5. Publikacja do Gov.pl
    Q_Publish -->|10. Consume| Go_Consumer
    Go_Consumer --> Data_Formatter
    Data_Formatter -->|11. Final POST| DaneGov
```
