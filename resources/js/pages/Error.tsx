import { Link } from 'inertia-adapter-solid'
import { useI18n } from '../lib/i18n'

export default function ErrorPage(props: { status: number }) {
  const { t, props: shared } = useI18n()
  return (
    <section class="hero">
      <p class="muted">{props.status}</p>
      <h1>{props.status === 404 ? t('error.not_found') : t('error.server')}</h1>
      <p><Link href={shared().site.homePath}>{t('error.back_home')}</Link></p>
    </section>
  )
}
