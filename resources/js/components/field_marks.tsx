import { For, Show } from 'solid-js'
import { useI18n } from '../lib/i18n'
import { stepsFor } from '../lib/steps'
import type { Activity } from '../types'
import { ScoreBadge } from './score_badge'
import { StepRow } from './step_row'

/** The pattern behind the score: the overall tempo, then one step row per measured component, then the evidence. */
export function FieldMarks(props: { activity: Activity }) {
  const { t, ago } = useI18n()
  const a = () => props.activity
  return (
    <section class="plate" aria-labelledby="activity-h">
      <h2 id="activity-h">{t('project.activity')}</h2>
      <div class="plate-score">
        <ScoreBadge score={a().score} tier={a().tier} size="lg" />
        <span class="muted small">{t('score.out_of')}</span>
      </div>
      <StepRow lit={stepsFor(a().score)} size="lg" animate />
      <Show when={a().cap}><p class="note small">{a().cap}</p></Show>
      <ul class="bars">
        <For each={a().components.filter(c => c.used)}>{c => (
          <li role="img" aria-label={`${t(`score.${c.name}`)}: ${Math.round(c.value * 100)}%`}>
            <div class="bar-label"><span>{t(`score.${c.name}`)}</span><span class="muted small">{Math.round(c.weight * 100)}%</span></div>
            <StepRow lit={stepsFor(c.value * 5)} size="sm" />
            <Show when={c.evidence}><span class="muted small">{c.evidence}</span></Show>
          </li>
        )}</For>
      </ul>
      <dl class="facts">
        <div><dt>{t('project.last_commit')}</dt><dd>{ago(a().lastCommitAt)}</dd></div>
        <div><dt>{t('project.last_release')}</dt><dd>{a().lastReleaseTag ? `${a().lastReleaseTag} · ${ago(a().lastReleaseAt)}` : t('common.never')}</dd></div>
        <div><dt>Commits 90d / 365d</dt><dd>{a().commits90} / {a().commits365}</dd></div>
        <Show when={a().allBranches}><div><dt>Branches (90d)</dt><dd>{a().activeBranches90} / {a().totalBranches}</dd></div></Show>
      </dl>
      <details class="how">
        <summary>{t('score.how')}</summary>
        <p class="small">{t('score.explain')}</p>
      </details>
    </section>
  )
}
