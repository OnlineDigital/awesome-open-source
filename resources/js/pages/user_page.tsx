import { Link, usePage } from 'inertia-adapter-solid'
import Snapshot from '../components/page_snapshot'

export default function UserPage() {
  const page = usePage<{ user: { id: string; name: string } }>()
  return <>
    <Snapshot name="Users/Show" title="One controller, one action, one file" description="The named users.show route passes the URL parameter to the show_user.go action in the users_controller folder. The controller sends props to the Solid Users/Show component." />
    <section class="demo-section">
      <h2>Parameter received by the server</h2>
      <dl class="counter-list"><div><dt>user.id</dt><dd id="user-id">{page.props.user.id}</dd></div><div><dt>user.name</dt><dd>{page.props.user.name}</dd></div></dl>
      <p>The ID is a route parameter rather than a user loaded from a database. Change the parameter and inspect the props returned by the controller.</p>
      <div class="controls"><Link href="/users/37">User 37</Link><Link href="/users/42">User 42</Link><Link href="/home">Back to demonstrations</Link></div>
    </section>
  </>
}
