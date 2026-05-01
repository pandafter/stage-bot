<script setup lang="ts">
import { ref, onMounted } from 'vue'

interface Post {
  id: string
  media_url: string
  caption?: string
  permalink: string
}

const posts = ref<Post[]>([])
const loading = ref(true)

const IG_PROFILE = 'https://instagram.com/scuderia_stage4'

onMounted(async () => {
  try {
    const res = await fetch('/api/instagram')
    if (res.ok) {
      const data = await res.json()
      posts.value = Array.isArray(data) ? data : []
    }
  } catch {
    // silently ignore — fallback shows below
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <section class="section ig-section" id="publicaciones">
    <div class="container">
      <span class="eyebrow">Instagram</span>
      <h2>Síguenos en<br /><span class="gold">Instagram</span></h2>

      <!-- Loading skeletons -->
      <div v-if="loading" class="ig-grid">
        <div v-for="i in 6" :key="i" class="ig-skeleton" />
      </div>

      <!-- Real posts from API -->
      <div v-else-if="posts.length > 0" class="ig-grid">
        <a
          v-for="post in posts"
          :key="post.id"
          :href="post.permalink"
          target="_blank"
          rel="noopener"
          class="ig-card"
        >
          <img :src="post.media_url" :alt="post.caption || 'Post de Scudería St4ge'" loading="lazy" />
          <div v-if="post.caption" class="ig-overlay">
            <p>{{ post.caption.slice(0, 120) }}{{ post.caption.length > 120 ? '…' : '' }}</p>
          </div>
        </a>
      </div>

      <!-- Fallback: no token configured or API error -->
      <div v-else class="ig-fallback">
        <div class="ig-fallback-icon">📸</div>
        <p class="ig-fallback-text">
          Mira nuestro contenido en Instagram — pista, karts y pilotos en acción.
        </p>
        <a :href="IG_PROFILE" target="_blank" rel="noopener" class="ig-fallback-link">
          @scuderia_stage4 →
        </a>
      </div>

      <div class="ig-cta">
        <a :href="IG_PROFILE" target="_blank" rel="noopener" class="btn-gold">
          Ver perfil completo →
        </a>
      </div>
    </div>
  </section>
</template>

<style scoped>
.ig-section { background: #060606; padding: 80px 0; }

.ig-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
  margin-top: 40px;
}
@media (max-width: 720px) {
  .ig-grid { grid-template-columns: repeat(2, 1fr); gap: 8px; }
}
@media (max-width: 460px) {
  .ig-grid { grid-template-columns: repeat(2, 1fr); gap: 6px; }
}

/* Card */
.ig-card {
  position: relative;
  aspect-ratio: 1;
  overflow: hidden;
  display: block;
  background: #111;
}
.ig-card img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform .4s ease;
}
.ig-card:hover img { transform: scale(1.06); }

.ig-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: flex-end;
  padding: 14px;
  background: linear-gradient(transparent 40%, rgba(0,0,0,0.88));
  opacity: 0;
  transition: opacity .25s;
}
.ig-card:hover .ig-overlay { opacity: 1; }
.ig-overlay p {
  font-size: 12px;
  color: rgba(255,255,255,0.85);
  line-height: 1.5;
  margin: 0;
}

/* Skeleton */
.ig-skeleton {
  aspect-ratio: 1;
  background: #111;
  animation: igpulse 1.2s ease-in-out infinite;
}
@keyframes igpulse {
  0%, 100% { opacity: 0.5; }
  50% { opacity: 0.9; }
}

/* Fallback */
.ig-fallback {
  margin-top: 40px;
  padding: 48px 24px;
  border: 1px dashed var(--border2);
  text-align: center;
}
.ig-fallback-icon { font-size: 40px; margin-bottom: 12px; }
.ig-fallback-text { font-size: 15px; color: var(--text-dim); margin-bottom: 16px; }
.ig-fallback-link {
  font-family: var(--font-display);
  font-weight: 800;
  font-size: 22px;
  text-transform: uppercase;
  letter-spacing: .06em;
  color: var(--gold);
}
.ig-fallback-link:hover { opacity: 0.8; }

/* CTA */
.ig-cta { text-align: center; margin-top: 32px; }
.ig-cta .btn-gold { font-size: 15px; padding: 14px 28px; }
</style>
