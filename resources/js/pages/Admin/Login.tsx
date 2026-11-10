import { useForm } from 'inertia-adapter-solid'
import { Field } from '../../components/admin_ui'

export default function Login() {
  const form = useForm({ email: '', password: '' })
  return (
    <form class="panel login" onSubmit={e => { e.preventDefault(); form.post('/admin/login') }}>
      <h1>Admin</h1>
      <Field label="Email" for="email" error={form.errors.email}>
        <input id="email" type="email" autocomplete="username" required value={form.data.email} onInput={e => form.setData('email', e.currentTarget.value)} />
      </Field>
      <Field label="Password" for="password">
        <input id="password" type="password" autocomplete="current-password" required value={form.data.password} onInput={e => form.setData('password', e.currentTarget.value)} />
      </Field>
      <button type="submit" disabled={form.processing}>Sign in</button>
    </form>
  )
}
