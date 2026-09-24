# nexus-interview

AI destekli teknik mülakat platformu. İş arayan kişilerin teknik mülakatlarını simüle ederek pratik yapmasını sağlar. Aynı zamanda AWS hands-on deneyimi kazanmak amacıyla bireysel bir eğitim projesidir.

---

## Amaç

- Adayların teknik mülakatlarını gerçekçi bir ortamda geliştirmesi
- AI ile sesli video mülakat deneyimi
- AWS ekosistemini uçtan uca kullanmak (altyapı, CI/CD, veritabanı, monitoring)
- Her bileşeni tak-çıkar tasarlamak: provider değişince sadece adapter değişir, iş mantığı dokunulmaz

---

## Tasarım Prensipleri

### SOLID

- **Single Responsibility:** Her modül tek bir işten sorumlu. `ai` paketi sadece prompt/response döngüsünü bilir, LLM provider'ı bilmez.
- **Open/Closed:** Yeni bir LLM veya STT provider eklemek için mevcut kodu değiştirmek gerekmez, sadece yeni bir adapter implement edilir.
- **Liskov Substitution:** Tüm provider'lar aynı interface'i karşılar. `GroqProvider` nerede kullanılıyorsa `OpenAIProvider` da sorunsuz çalışır.
- **Interface Segregation:** Büyük interface'ler yok. `Transcriber`, `Synthesizer`, `LLMClient` ayrı ayrı tanımlanır.
- **Dependency Inversion:** İş mantığı concrete type'a bağımlı değil, interface'e bağımlı. Provider'lar dışarıdan inject edilir.

### Tak-Çıkar Mimari

Her harici araç (LLM, STT, TTS, e-posta) bir interface arkasına alınır. Provider değişince sadece ilgili adapter dosyası değişir.

```
internal/ai/
  port.go          ← LLMClient interface (iş mantığı bunu görür)
  service.go       ← iş mantığı, sadece interface'e bağımlı
  provider/
    groq.go        ← Groq API adapter
    openai.go      ← OpenAI adapter (ileride)

internal/speech/
  port.go          ← Transcriber, Synthesizer interface'leri
  service.go       ← iş mantığı
  provider/
    transcribe.go  ← AWS Transcribe adapter
    polly.go       ← AWS Polly adapter
```

---

## Teknoloji Stack

### Backend
- **Go** — ana backend dili
- **Modern Monolit** mimarisiyle başlanır, mikroservise geçişe uyumlu modüler yapı
- Her modül kendi interface boundary'sine sahip olur (doğrudan struct paylaşımı yok)

### Frontend
- **Next.js** (React) — aday mülakat ekranı + admin panel

### Veritabanı
- **AWS RDS (PostgreSQL)** — kullanıcılar, mülakatlar, sorular, değerlendirmeler
- **MongoDB** — ilerleyen fazlarda gerektiğinde (mülakat transkript logları, esnek döküman yapıları)

### AI / LLM
- **Groq API (Llama 3.3 70B)** — saf HTTP API call'ları ile entegrasyon, geliştirme boyunca ücretsiz tier
- Inference engineering (self-hosting, quantization vb.) sonraki fazlarda değerlendirilecek
- Provider değişimi tek bir adapter dosyası değiştirmekten ibaret olacak şekilde tasarlanır

### Ses / Konuşma
- **AWS Transcribe** — gerçek zamanlı konuşma → metin (STT)
- **AWS Polly** — metin → ses, AI interviewer sesi (TTS)

### AWS Altyapı (Manuel Kurulum)
- **ECS Fargate** — container orchestration
- **RDS PostgreSQL** — ilişkisel veri
- **S3** — video kayıtları, ses dosyaları, statik dosyalar
- **CodePipeline + CodeBuild** — CI/CD
- **ECR** — Docker image registry
- **CloudWatch + Grafana** — log ve metrik izleme
- **SES** — aday davet e-postaları
- **VPC, ALB, Security Groups** — ağ ve güvenlik katmanı

> Tüm AWS kaynakları AWS Console veya CLI üzerinden manuel olarak kurulur. IaC (Terraform) şimdilik kullanılmaz.

### Monitoring
- **Grafana** — dashboard
- **Prometheus** — metrik toplama
- **CloudWatch** — AWS native log/alarm

---

## Mimari: Modern Monolit (Mikroservise Hazır)

```
nexus-interview/
├── cmd/
│   └── server/          # main.go — tek binary, dependency injection root'u
├── internal/
│   ├── interview/        # mülakat akışı, oturum yönetimi
│   ├── question/         # soru setleri, soru bankası
│   ├── candidate/        # aday profili, davet sistemi
│   ├── evaluation/       # AI değerlendirme, scoring
│   ├── speech/           # STT/TTS port + provider'ları
│   ├── ai/               # LLM port + provider'ları
│   ├── admin/            # admin panel API
│   ├── auth/             # JWT, session yönetimi
│   └── notification/     # e-posta (SES)
├── pkg/                  # paylaşılan yardımcı paketler (config, logger, db)
├── deployments/
│   └── docker/           # Dockerfile'lar
└── docs/                 # API dokümantasyonu
```

Her `internal/` modülü:
- `port.go` — modülün dışarıya sunduğu ve içeride kullandığı interface'ler
- `service.go` — iş mantığı (sadece interface'lere bağımlı)
- `provider/` — harici servis adapter'ları (Groq, AWS vb.)
- `repository.go` — veritabanı erişim katmanı (yine interface arkasında)

---

## Mülakat Akışı

```
İK / Admin
  │
  ├── İlan oluştur
  ├── Soru seti yükle (yazılı veya ses)
  └── Aday davet et (SES ile e-posta linki)
        │
        ▼
Aday (tarayıcı üzerinden)
  │
  ├── Linke tıkla → kimlik doğrulama (link tabanlı, şifresiz)
  ├── Kamera + mikrofon izni
  ├── Video görüşme başlar
  │     ├── AI soruyu sesli sorar (Polly → TTS)
  │     ├── Aday yanıt verir (ses → Transcribe → STT → metin)
  │     ├── Ekranda aday konuşurken transcript görünür (altyazı)
  │     ├── LLM (Groq API) yanıtı analiz eder, sıradaki soruyu belirler
  │     └── Video kaydı S3'e yüklenir
  └── Görüşme biter → değerlendirme raporu oluşturulur

Admin
  └── Raporu görüntüler (scoring, transkript, video)
```

---

## Geliştirme Sırası

Backend önce, frontend sonra. Backend tamamen tamamlanmadan frontend'e geçilmez.

- **Backend** — Go monolit, tüm API'lar, iş mantığı, veritabanı katmanı
- **AWS altyapısı** — RDS, S3, SES, Transcribe, Polly gibi servisler kullanıcı tarafından manuel kurulur; ajan AWS kaynaklarını provision etmez, yönetmez veya AWS CLI/SDK komutları çalıştırmaz
- **Frontend** — backend API'ları hazır olduktan sonra başlanır

---

## Faz Planı

### Faz 1 — Temel Altyapı (Hafta 1-3)
- [ ] AWS Console/CLI ile VPC, RDS, S3, ECR manuel kurulumu
- [ ] Go monolit iskelet (HTTP server, routing, middleware)
- [ ] Auth modülü (admin JWT, aday magic link)
- [ ] Admin panel API (CRUD: ilanlar, soru setleri, adaylar)
- [ ] Next.js admin panel UI (temel CRUD ekranları)
- [ ] CodePipeline + CodeBuild CI/CD pipeline

### Faz 2 — Mülakat Motoru (Hafta 4-7)
- [ ] WebRTC / WebSocket tabanlı görüşme odası
- [ ] AWS Transcribe gerçek zamanlı STT entegrasyonu
- [ ] AWS Polly TTS entegrasyonu (AI ses)
- [ ] LLM entegrasyonu — Groq API, saf HTTP client (tak-çıkar adapter)
- [ ] Mülakat akışı orkestrasyon (soru → cevap → sonraki soru)
- [ ] Video kaydı → S3 yükleme

### Faz 3 — Değerlendirme & Raporlama (Hafta 8-10)
- [ ] AI tabanlı cevap değerlendirme ve scoring (Groq API)
- [ ] Mülakat transkript raporu
- [ ] Admin değerlendirme ekranı (video replay + transkript)
- [ ] SES ile aday davet ve sonuç bildirimi
- [ ] Grafana + CloudWatch monitoring kurulumu

### Faz 4 — Inference Engineering (Kapalı — Sonraya)
> Şimdilik değerlendirilmeyecek. Groq API ile ilerlenir.
> Gerektiğinde tak-çıkar adapter mimarisi sayesinde kolayca eklenebilir.
- [ ] EC2 g4dn spot instance kurulumu
- [ ] Ollama veya vLLM ile Llama 3 self-hosting
- [ ] Groq → kendi modelimize geçiş
- [ ] Model optimizasyon deneyleri (quantization, batching)
- [ ] Latency ve maliyet karşılaştırması (Groq vs self-hosted)

### Faz 5 — Mikroservis Geçişi (Hafta 11+)
- [ ] Servis sınırlarını belirle (yük ve bağımlılık analizine göre)
- [ ] İlk ayrılacak servis: `speech-service` (STT/TTS — en bağımsız)
- [ ] `evaluation-service` ayrımı
- [ ] Servisler arası iletişim (gRPC veya message queue)
- [ ] Her servis için bağımsız deployment pipeline

---

## Kod Yapısı: Interface Örnekleri

```go
// internal/ai/port.go
package ai

type LLMClient interface {
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}

// internal/speech/port.go
package speech

type Transcriber interface {
    Transcribe(ctx context.Context, audio io.Reader) (string, error)
}

type Synthesizer interface {
    Synthesize(ctx context.Context, text string) (io.Reader, error)
}

// internal/notification/port.go
package notification

type Mailer interface {
    Send(ctx context.Context, msg Message) error
}
```

Provider'lar bu interface'leri implemente eder. `service.go` katmanları asla concrete type görmez.

```go
// cmd/server/main.go — dependency injection root
groqClient := groq.NewClient(cfg.GroqAPIKey)   // LLMClient interface'ini karşılar
aiService   := ai.NewService(groqClient)         // sadece LLMClient alır
```

---

## Maliyet Stratejisi (200$ AWS Kredi)

| Kaynak | Kullanım | Tahmini Maliyet |
|---|---|---|
| RDS db.t3.micro | 7/24 | ~$15/ay |
| ECS Fargate | düşük trafik | ~$10/ay |
| S3 | video/ses depolama | ~$5/ay |
| CloudWatch | loglar | ~$5/ay |
| SES | e-posta | ~$1/ay |
| **LLM** | **Groq free tier** | **$0** |

> EC2 GPU inference deneyleri ileriki bir fazda değerlendirilecek. Şu an kapsam dışı.

---

## Öğrenme Hedefleri

- Go ile production-grade REST + WebSocket API
- SOLID prensipleri ve tak-çıkar adapter mimarisi
- Modern monolit mimarisi ve servis sınırı tasarımı
- AWS ekosistemi hands-on (ECS, RDS, S3, SES, Transcribe, Polly, CodePipeline)
- Mikroservis geçiş stratejisi ve pattern'leri
- Gerçek zamanlı ses/video akışı entegrasyonu

---

## Repo Yapısı (Başlangıç)

```
nexus-interview/
├── README.md
├── .github/
│   └── workflows/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── interview/
│   │   ├── port.go
│   │   ├── service.go
│   │   └── repository.go
│   ├── question/
│   │   ├── port.go
│   │   ├── service.go
│   │   └── repository.go
│   ├── candidate/
│   │   ├── port.go
│   │   ├── service.go
│   │   └── repository.go
│   ├── evaluation/
│   │   ├── port.go
│   │   ├── service.go
│   │   └── repository.go
│   ├── speech/
│   │   ├── port.go
│   │   ├── service.go
│   │   └── provider/
│   │       ├── transcribe.go
│   │       └── polly.go
│   ├── ai/
│   │   ├── port.go
│   │   ├── service.go
│   │   └── provider/
│   │       └── groq.go
│   ├── admin/
│   │   ├── port.go
│   │   └── handler.go
│   ├── auth/
│   │   ├── port.go
│   │   └── service.go
│   └── notification/
│       ├── port.go
│       ├── service.go
│       └── provider/
│           └── ses.go
├── pkg/
│   ├── config/
│   ├── logger/
│   └── database/
├── deployments/
│   └── docker/
│       └── Dockerfile
└── docs/
    └── architecture.md
```
