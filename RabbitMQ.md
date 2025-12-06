# 🐇 Konfiguracja RabbitMQ

System wykorzystuje **Topic Exchange** o nazwie `lost-found.events`.

## Kolejki i Routing Keys

| Kolejka (Queue)        | Routing Key       | Nadawca             | Odbiorca              | Opis                                                 |
| ---------------------- | ----------------- | ------------------- | --------------------- | ---------------------------------------------------- |
| `q.lost-items.ingest`  | `item.submitted`  | Service A (Gateway) | Service B (Python)    | Surowe dane zgłoszenia + URL zdjęcia                 |
| `q.lost-items.publish` | `item.vectorized` | Service B (Python)  | Service C (Publisher) | Dane wzbogacone o ID wektora, gotowe do formatowania |

## Schemat przepływu komunikatów

1. **Service A → RabbitMQ**: Po zatwierdzeniu formularza przez urzędnika, Gateway publikuje wiadomość z routing key `item.submitted` do exchange'a `lost-found.events`
2. **RabbitMQ → Service B**: Wiadomość trafia do kolejki `q.lost-items.ingest`, gdzie Python Worker ją konsumuje
3. **Service B → RabbitMQ**: Po wygenerowaniu embeddingów i zapisie w Qdrant, Python Worker publikuje wiadomość z routing key `item.vectorized`
4. **RabbitMQ → Service C**: Wiadomość trafia do kolejki `q.lost-items.publish`, gdzie Publisher Go ją konsumuje i wysyła do dane.gov.pl
