import { Link as InertiaLink } from 'inertia-adapter-solid'
import { type ComponentProps, mergeProps } from 'solid-js'

/** Inertia Link that preloads the target page when the pointer hovers (or the link is focused). Pass `prefetch={false}` to opt out. */
export function Link(props: ComponentProps<typeof InertiaLink>) {
  return <InertiaLink {...mergeProps({ prefetch: 'hover' as const }, props)} />
}
