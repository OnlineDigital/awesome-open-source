import { Link } from '../components/link'
import { StepRow } from '../components/step_row'
import { useI18n } from '../lib/i18n'

export default function ErrorPage(props: { status: number }) {
  const { t, props: shared } = useI18n()
  return (
    <section class="wrap main"><div class={`error-page`}>
      <p class="error-code">{props.status}</p>
      <StepRow lit={props.status === 404 ? 1 : 3} size="lg" />
      <h1>{props.status === 404 ? t('error.not_found') : t('error.server')}</h1>
      <p><Link href={shared().site.homePath} class="button">{t('error.back_home')}</Link></p>
    </div></section>
  )
}
