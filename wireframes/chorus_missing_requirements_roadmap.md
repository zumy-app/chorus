# Chorus Production Readiness & Phase 2/3 Roadmap

## 1. Missing Production-Ready Messaging Features (Gap Analysis)
To compete with WhatsApp and Telegram, Chorus needs the following essential messaging utilities:

### A. Message Management
- **Read Receipts & Delivery Status:** Visual indicators (single/double ticks) for sent, delivered, and read.
- **Message Actions:** Long-press to Reply, Forward, Delete (for me/everyone), and Pin.
- **Search:** Universal search for messages and media within chats.
- **Archive & Mute:** Ability to hide or silence specific conversations.

### B. Media & File Handling
- **Media Gallery:** A centralized view of all photos, videos, and links shared in a specific chat.
- **Document Sharing:** Support for PDFs, docs, and spreadsheets (critical for teacher-student homework).
- **Location Sharing:** Real-time or static location sharing.

### C. Privacy & Security
- **Block & Report:** Standard safety features for a marketplace environment.
- **Two-Factor Authentication (2FA):** Enhancing the WhatsApp OTP login.
- **Privacy Settings:** Last seen, profile photo visibility, and status controls.

---

## 2. Phase 2: Audio Calling with Smart Captions
The goal is to turn calls into active learning moments.
- **Real-time Transcription:** Live text overlays for both speakers.
- **Interactive Translation:** Tap any captioned word to see a translation without interrupting the call.
- **Scrollable Transcript:** A vertical, scrollable side or overlay panel to review previous parts of the conversation while still live.
- **Vocabulary Capture:** A "Save" button next to transcribed phrases to instantly add them to the SRS queue.

---

## 3. Phase 3: Video Calling & Desktop Optimization
- **Dual-View UI:** Teacher/Partner on main screen, learner in PiP (Picture-in-Picture).
- **Screen Sharing:** Critical for teachers to present slides or interactive drills.
- **Immersive Captions:** Subtitle-style overlays that don't obscure faces.
- **Desktop Dashboard:** A "Control Center" for the web app where calls, chat, and learning data coexist.

---

## 4. Technical Architecture (Level 4 Load Balancer + Redis Scaling)
- **L4 LB:** Directs TCP/UDP traffic to the least-connected chat server instance.
- **Redis Pub/Sub:** Handles the user-to-server mapping and cross-server message routing.
- **Persistence:** Messages are committed to a distributed DB (Postgres/Mongo) before Redis routing to ensure no data loss during delivery.
