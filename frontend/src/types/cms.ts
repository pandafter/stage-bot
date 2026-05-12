export interface CmsSection {
  tenant_id: string
  section_key: string
  data: Record<string, unknown>
  is_published: boolean
  version: number
  updated_at: string
}

export interface HeroData {
  title: string
  subtitle: string
  cta_text: string
  cta_href: string
  bg_media_id?: number
  bg_url?: string
}

export interface StatItem {
  icon: string
  value: string
  label: string
}

export interface StatsData {
  items: StatItem[]
}

export interface InstructorItem {
  name: string
  role: string
  bio_html: string
  photo_media_id?: number
  photo_url?: string
}

export interface InstructoresData {
  items: InstructorItem[]
}

export interface FaqItem {
  q: string
  a_html: string
}

export interface FaqData {
  items: FaqItem[]
}

export interface GalleryData {
  media_ids: number[]
  urls?: string[]
}

export interface CtaData {
  title: string
  subtitle: string
  cta_text: string
  cta_href: string
  bg_media_id?: number
  bg_url?: string
}

export interface Theme {
  primary_color: string
  secondary_color: string
  accent_color: string
  font_heading: string
  font_body: string
  logo_url: string
  favicon_url: string
}
