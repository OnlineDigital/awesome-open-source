import { createSignal } from 'solid-js'
import { useI18n } from '../lib/i18n'
import { IconPlay } from './icons'

// Click-to-load facade: no YouTube JavaScript or cookies until the visitor presses play.
export function YouTube(props: { id: string; thumb: string; embed: string; title: string }) {
  const { t } = useI18n()
  const [playing, setPlaying] = createSignal(false)
  return (
    <div class="video">
      {playing() ? (
        <iframe src={`${props.embed}?autoplay=1&rel=0`} title={props.title} allow="accelerometer; autoplay; encrypted-media; picture-in-picture; fullscreen" allowfullscreen referrerpolicy="strict-origin-when-cross-origin" />
      ) : (
        <button type="button" class="video-play" onClick={() => setPlaying(true)} aria-label={`${t('project.play')}: ${props.title}`}>
          <img src={props.thumb} alt="" loading="lazy" width="480" height="360" />
          <span class="video-icon" aria-hidden="true"><IconPlay width={28} height={28} /></span>
        </button>
      )}
    </div>
  )
}
