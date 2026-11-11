/** The 16-step scale: score / 5 mapped onto sixteen keys, coloured in quarters (1 pink, 2 salmon, 3 yellow, 4 mint). */
export const STEPS = 16
export const stepsFor = (score: number | null | undefined) => (score == null ? 0 : Math.max(0, Math.min(STEPS, Math.round((score / 5) * STEPS))))
export const quarterOf = (steps: number) => Math.max(1, Math.min(4, Math.ceil(steps / 4)))
export const quarterForScore = (score: number | null | undefined) => quarterOf(stepsFor(score))
