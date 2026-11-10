import { Link } from 'inertia-adapter-solid'
import { For, Show } from 'solid-js'
import { Gallery } from '../components/gallery'
import { ProjectCard } from '../components/project_card'
import { ScoreBreakdown } from '../components/score_breakdown'
import { YouTube } from '../components/youtube'
import { useI18n } from '../lib/i18n'
import type { Activity, Card, Project } from '../types'

export default function ProjectPage(props: { project: Project; activity: Activity | null; related: Card[]; video: { id: string; thumb: string; embed: string }; markdownPath: string }) {
  const { t, number, props: shared } = useI18n()
  const p = () => props.project
  const catPath = () => (shared().locale === shared().site.defaultLocale ? '' : `/${shared().locale}`) + `/category/${p().categorySlug}`
  return (
    <>
      <nav class="crumbs" aria-label="Breadcrumb">
        <Link href={shared().site.homePath}>{shared().site.name}</Link>
        <Show when={p().categorySlug}> <span aria-hidden="true">/</span> <Link href={catPath()}>{p().categoryName}</Link></Show>
        {' '}<span aria-hidden="true">/</span> <span>{p().repo}</span>
      </nav>
      <header class="project-head">
        {p().avatarUrl && <img class="avatar avatar-lg" src={p().avatarUrl} alt="" width="64" height="64" />}
        <div>
          <h1>{p().repo}</h1>
          <p class="muted">{p().fullName}</p>
        </div>
      </header>
      <p class="lead">{p().tagline}</p>
      <div class="actions">
        <a class="button" href={`https://github.com/${p().fullName}`} rel="noopener" target="_blank">{t('project.github')} ↗</a>
        <Show when={p().homepage}><a class="button secondary" href={p().homepage} rel="noopener" target="_blank">{t('project.homepage')} ↗</a></Show>
        <a class="button secondary" href={props.markdownPath} type="text/markdown">MD</a>
        <span class="muted">★ {number(p().stars)}</span>
      </div>

      <div class="project-layout">
        <div class="project-main">
          <Show when={props.video.id}>
            <section aria-labelledby="demo-h"><h2 id="demo-h">{t('project.demo')}</h2><YouTube id={props.video.id} thumb={props.video.thumb} embed={props.video.embed} title={`${t('project.demo')}: ${p().repo}`} /></section>
          </Show>
          <Show when={p().description}>
            <section class="prose">
              <For each={p().description.split(/\n{2,}/)}>{para => <p>{para}</p>}</For>
              <Show when={p().machineTranslated}><p class="muted small">{t('project.machine_translated')}</p></Show>
            </section>
          </Show>
          <Show when={(p().screenshots?.length ?? 0) > 0}>
            <section aria-labelledby="shots-h"><h2 id="shots-h">{t('project.screenshots')}</h2><Gallery shots={p().screenshots!} fallbackAlt={`${p().repo} screenshot`} /></section>
          </Show>
        </div>
        <aside class="project-side">
          <Show when={props.activity}>{a => <ScoreBreakdown activity={a()} />}</Show>
          <dl class="panel facts">
            <Show when={p().license}><div><dt>{t('project.license')}</dt><dd>{p().license}</dd></div></Show>
            <Show when={p().language}><div><dt>{t('project.language')}</dt><dd>{p().language}</dd></div></Show>
            <Show when={p().topics.length > 0}><div><dt>{t('project.topics')}</dt><dd class="pill-list"><For each={p().topics}>{tp => <Link class="pill" href={`${shared().locale === shared().site.defaultLocale ? '' : '/' + shared().locale}/search?q=${encodeURIComponent(tp)}`}>{tp}</Link>}</For></dd></div></Show>
          </dl>
        </aside>
      </div>

      <Show when={props.related.length > 0}>
        <section aria-labelledby="rel-h"><h2 id="rel-h">{t('project.related')}</h2><div class="grid"><For each={props.related}>{r => <ProjectCard project={r} />}</For></div></section>
      </Show>
    </>
  )
}
