# 🐇 Konfiguracja RabbitMQ

System wykorzystuje **Topic Exchange** o nazwie `lost-found.events`.

## Kolejki i Routing Keys

| Kolejka (Queue)        | Routing Key       | Nadawca             | Odbiorca       | Opis                                                        |
| ---------------------- | ----------------- | ------------------- | -------------- | ----------------------------------------------------------- |
| `q.lost-items.embed`  | `item.submitted`  | Service A (Gateway) | Clip Service | Surowe dane zgłoszenia (description, category)        |
| `q.lost-items.injgst` | `item.embedded` | Clip Service      | Qdrant Service      | Dane są zembbedowane i gotowe do zapisu w Qdrant |
| `q.lost-items.publish` |`item.vectorized` | Qdrant Service | Service C (Publisher) | Dane z vector_id po zapisie w Qdrant, gotowe do publikacji |

## Schemat przepływu komunikatów

1. **Service A → RabbitMQ**: Po zatwierdzeniu formularza przez urzędnika, Gateway publikuje wiadomość z routing key `item.submitted` do exchange'a `lost-found.events`
2. **RabbitMQ → Clip Service**: Wiadomość trafia do kolejki `q.lost-items.embed`, gdzie Clip Service ją konsumuje, generuje embeddingi.
3. **Clip Service → RabbitMQ**: Po wygenerowaniu embeddingu Clip Service publikuje wiadomość z routing key `item.ingest` zawierającą embedding dla Qrdant
4. **RabbitMQ → Qrdant Service**: Wiadomość trafia do `q.lost-items.ingest`. Embedding zostaje wykorzystany do osadzenia w Qrdant jako wektor.
5. **Qrdant Service → RabbitMQ**:
Publikuje wiadomość z routing key `item.vectorized`.
4. **RabbitMQ → Publisher**: Wiadomość trafia do kolejki `q.lost-items.publish`, gdzie Publisher ją konsumuje i publikuje do `dane.gov.pl`.
