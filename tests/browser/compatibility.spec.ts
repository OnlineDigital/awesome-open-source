import {test, expect, type Page} from '@playwright/test'
const pause = (page:Page,ms:number) => page.waitForTimeout(ms)
async function open(page:Page,path='/home') {
  const errors:Error[]=[]
  page.on('pageerror',error=>errors.push(error))
  await page.goto(path)
  if(errors.length) throw errors[0]
  await page.waitForFunction(()=>window.__lab?.ready)
}
async function props(page:Page) {return JSON.parse((await page.locator('#props').textContent())!)}
async function stats(page:Page) {return (await page.request.get('/stats')).json()}
async function requests(page:Page,match:string) {return (await stats(page)).requests.filter((r:any)=>r.path===match)}
async function visit(page:Page,url:string,options:any={}) {
  await page.evaluate(({url,options})=>new Promise<void>(resolve=>window.__lab.router.visit(url,{...options,onFinish:()=>resolve()})),{url,options})
}
async function reload(page:Page,options:any={}) {
  await page.evaluate(options=>new Promise<void>(resolve=>window.__lab.router.reload({...options,onFinish:()=>resolve()})),options)
}
test.beforeEach(async({request})=>{await request.post('/reset')})

test('F05 initial HTML, production assets, SPA navigation and back/forward',async({page})=>{
  await open(page); const boot=await page.evaluate(()=>window.__lab.boot)
  await page.locator('#nav-other').click(); await expect(page.locator('#component')).toHaveText('Other')
  expect(await page.evaluate(()=>window.__lab.boot)).toBe(boot)
  await page.goBack(); await expect(page.locator('#component')).toHaveText('Home')
  await page.goForward(); await expect(page.locator('#component')).toHaveText('Other')
})
test('F01 hover under 75ms makes no request',async({page})=>{
  await open(page); await page.locator('#hover').dispatchEvent('mouseenter'); await pause(page,25)
  await page.locator('#hover').dispatchEvent('mouseleave'); await pause(page,150)
  expect(await requests(page,'/other?case=hover')).toHaveLength(0)
})
test('F01 hover prefetch exactly once, Purpose header, cached navigation',async({page})=>{
  await open(page); await page.locator('#hover').hover()
  await expect.poll(async()=> (await requests(page,'/other?case=hover')).length).toBe(1)
  await expect.poll(()=>page.evaluate(()=>!!window.__lab.router.getCached('/other?case=hover'))).toBe(true)
  expect((await requests(page,'/other?case=hover'))[0].purpose).toBe('prefetch')
  await page.locator('#hover').click(); await expect(page.locator('#component')).toHaveText('Other')
  expect((await props(page)).label).toBe('other'); expect(await requests(page,'/other?case=hover')).toHaveLength(1)
  expect(await page.evaluate(()=>window.__lab.events)).toEqual(expect.arrayContaining(['prefetching','prefetched','navigate']))
})
test('F01 global hover delay',async({page})=>{
  await open(page); await page.evaluate(()=>window.__lab.config.set('prefetch.hoverDelay',250))
  await page.locator('#hover').hover(); await pause(page,120); expect(await requests(page,'/other?case=hover')).toHaveLength(0)
  await expect.poll(async()=> (await requests(page,'/other?case=hover')).length).toBe(1)
})
test('F01 per-link cache expires and navigation fetches again',async({page})=>{
  await open(page); await page.locator('#expiry').hover()
  await expect.poll(()=>page.evaluate(()=>!!window.__lab.router.getCached('/other?case=expiry'))).toBe(true)
  await page.mouse.move(950,700); await pause(page,350)
  await visit(page,'/other?case=expiry'); await expect(page.locator('#component')).toHaveText('Other')
  expect(await requests(page,'/other?case=expiry')).toHaveLength(2)
})
test('F01 global cache lifetime',async({page})=>{
  await open(page,'/home?cache=200')
  await page.locator('#global-cache').hover(); await expect.poll(()=>page.evaluate(()=>!!window.__lab.router.getCached('/other?case=global'))).toBe(true)
  await pause(page,300); await visit(page,'/other?case=global')
  expect(await requests(page,'/other?case=global')).toHaveLength(2)
})
test('F01 click strategy prefetches on mousedown',async({page})=>{
  await open(page); await page.locator('#click-prefetch').dispatchEvent('mousedown',{button:0})
  await expect.poll(()=>page.evaluate(()=>!!window.__lab.router.getCached('/other?case=click'))).toBe(true)
  await page.locator('#click-prefetch').dispatchEvent('mouseup',{button:0}); await expect(page.locator('#component')).toHaveText('Other')
  expect(await requests(page,'/other?case=click')).toHaveLength(1)
})
test('F01 mount and combined strategies deduplicate',async({page})=>{
  await open(page); await page.locator('#mount-links').click()
  for(const name of ['mount','combined']) await expect.poll(()=>page.evaluate(name=>!!window.__lab.router.getCached('/other?case='+name),name)).toBe(true)
  await page.locator('#combined').hover(); await pause(page,200)
  expect(await requests(page,'/other?case=combined')).toHaveLength(1)
  expect(await requests(page,'/other?case=mount')).toHaveLength(1)
})
test('F01 programmatic prefetch and explicit tag invalidation',async({page})=>{
  await open(page); await page.evaluate(()=>window.__lab.router.prefetch('/other?case=programmatic',{}, {cacheTags:['users']}))
  await expect.poll(()=>page.evaluate(()=>!!window.__lab.router.getCached('/other?case=programmatic'))).toBe(true)
  await page.evaluate(()=>window.__lab.router.flushByCacheTags('users'))
  expect(await page.evaluate(()=>window.__lab.router.getCached('/other?case=programmatic'))).toBeNull()
  await visit(page,'/other?case=programmatic');expect(await requests(page,'/other?case=programmatic')).toHaveLength(2)
})
test('F01 mutation invalidates tagged prefetch cache',async({page})=>{
  await open(page); await page.locator('#tagged').hover(); await expect.poll(()=>page.evaluate(()=>!!window.__lab.router.getCached('/other?case=tagged'))).toBe(true)
  await visit(page,'/submit',{method:'post',data:{name:'Ada'},invalidateCacheTags:['users']})
  await expect(page.locator('#component')).toHaveText('FormPage')
  expect(await page.evaluate(()=>window.__lab.router.getCached('/other?case=tagged'))).toBeNull()
})
test('F01 usePrefetch state and flush on current page',async({page})=>{
  await open(page); await page.evaluate(()=>window.__lab.router.prefetch('/other'))
  await expect.poll(()=>page.evaluate(()=>!!window.__lab.router.getCached('/other'))).toBe(true)
  await visit(page,'/other')
  await expect.poll(()=>page.evaluate(()=>window.__lab.prefetch.isPrefetched)).toBe(true)
  expect(await page.evaluate(()=>window.__lab.prefetch.lastUpdatedAt)).toBeGreaterThan(0)
  await page.evaluate(()=>window.__lab.prefetch.flush())
  expect(await page.evaluate(()=>window.__lab.router.getCached('/other'))).toBeNull()
})
test('F01 rapid hover/unhover deduplicates',async({page})=>{
  await open(page)
  for(let i=0;i<4;i++){await page.locator('#hover').dispatchEvent('mouseenter');await pause(page,15);await page.locator('#hover').dispatchEvent('mouseleave')}
  expect(await requests(page,'/other?case=hover')).toHaveLength(0)
  await page.locator('#hover').hover(); await expect.poll(()=>page.evaluate(()=>!!window.__lab.router.getCached('/other?case=hover'))).toBe(true)
  expect(await requests(page,'/other?case=hover')).toHaveLength(1)
})

test('F02 only users avoids excluded callbacks and merges existing props',async({page})=>{
  await open(page,'/props'); const before=(await stats(page)).evaluations
  await reload(page,{only:['users']}); const after=(await stats(page)).evaluations
  expect(after.users).toBe(before.users+1); expect(after.companies).toBe(before.companies)
  const p=await props(page);expect(p.users).toEqual([{id:1,name:'Ada'}]);expect(p.companies).toEqual(['Acme']);expect(p.always).toBe('always')
})
test('F02 except companies avoids callback',async({page})=>{
  await open(page,'/props');const before=(await stats(page)).evaluations
  await reload(page,{except:['companies']});const after=(await stats(page)).evaluations
  expect(after.companies).toBe(before.companies);expect(after.users).toBe(before.users+1)
})
test('F02 optional initially absent and requested explicitly',async({page})=>{
  await open(page,'/props');expect((await props(page)).optional).toBeUndefined();expect((await stats(page)).evaluations.optional??0).toBe(0)
  await reload(page,{only:['optional']});expect((await props(page)).optional).toBe('optional-value')
})
test('F02 nested only auth.notifications',async({page})=>{
  await open(page,'/props');const before=(await stats(page)).evaluations
  await reload(page,{only:['auth.notifications']});const after=(await stats(page)).evaluations
  expect(after['auth.notifications']).toBe(before['auth.notifications']+1);expect(after.users).toBe(before.users)
  expect((await props(page)).auth.notifications).toEqual(['hello'])
})
test('F02 link-level only selected callback',async({page})=>{
  await open(page,'/props');const before=(await stats(page)).evaluations
  await page.locator('#only-link').click();await expect.poll(async()=> (await stats(page)).evaluations.users).toBe(before.users+1)
  expect((await stats(page)).evaluations.companies).toBe(before.companies)
})
test('F06 reload preserves local state and scroll',async({page})=>{
  await open(page,'/props');await page.locator('#local-state').fill('retained');await page.evaluate(()=>window.scrollTo(0,600));await reload(page,{only:['users']})
  await expect(page.locator('#local-state')).toHaveValue('retained');expect(await page.evaluate(()=>window.scrollY)).toBe(600)
})
test('F03 deferred fallback then content and parallel groups',async({page})=>{
  let initial:any
  await open(page)
  const version=await page.evaluate(()=>window.__lab.page().version)
  const response=await page.request.get('/deferred',{headers:{'X-Inertia':'true','X-Inertia-Version':version}});initial=await response.json()
  expect(initial.props.slow).toBeUndefined();expect(initial.deferredProps.default).toContain('slow');expect(initial.deferredProps.extra).toContain('second')
  await open(page,'/deferred');await expect(page.locator('#slow')).toHaveText('slow-value');await expect(page.locator('#second')).toHaveText('second-value')
  expect((await stats(page)).evaluations.slow).toBe(1)
})
test('F03 deferred fallback is visible before delayed resolution',async({page})=>{
  await page.route('**/deferred',async route=>{if(route.request().headers()['x-inertia'])await new Promise(r=>setTimeout(r,400));await route.continue()})
  await open(page,'/deferred');await expect(page.locator('#fallback')).toBeVisible();await expect(page.locator('#slow')).toHaveText('slow-value')
})
test('F04 once callback reused across eligible navigation',async({page})=>{
  await open(page,'/once');expect((await stats(page)).evaluations.lookup).toBe(1)
  await visit(page,'/once-other');expect((await stats(page)).evaluations.lookup).toBe(1);expect((await props(page)).lookup).toEqual(['US','RO'])
})
test('F04 explicit only refreshes once',async({page})=>{
  await open(page,'/once');await reload(page,{only:['lookup']});expect((await stats(page)).evaluations.lookup).toBe(2)
})
test('F04 server forced fresh once',async({page})=>{
  await open(page,'/once');await visit(page,'/once?fresh=1');expect((await stats(page)).evaluations.lookup).toBe(2)
})
test('F04 once forgotten after leaving section',async({page})=>{
  await open(page,'/once');await visit(page,'/other');await visit(page,'/once');expect((await stats(page)).evaluations.lookup).toBe(2)
})
test('F07 persistent layout retains counter',async({page})=>{
  await open(page);await page.locator('#layout-counter').click();await visit(page,'/other');await expect(page.locator('#layout-counter')).toHaveText('1')
})
test('F08 shared namespaced data survives partial reload',async({page})=>{
  await open(page,'/props');expect((await props(page)).shared.app).toBe('compat-lab');await reload(page,{only:['users']});expect((await props(page)).shared.app).toBe('compat-lab')
})
test('F09 stale asset version refreshes full page',async({page})=>{
  await open(page);const boot=await page.evaluate(()=>window.__lab.boot)
  await page.route('**/other',route=>route.continue({headers:{...route.request().headers(),'x-inertia-version':'old'}}))
  await page.evaluate(()=>window.__lab.router.visit('/other'))
  await expect(page.locator('#component')).toHaveText('Other');expect(await page.evaluate(()=>window.__lab.boot)).not.toBe(boot)
})
test('F10 title and metadata replaced without duplicates',async({page})=>{
  await open(page);await expect(page).toHaveTitle('Home — inertia-go · Solid template');await visit(page,'/other');await expect(page).toHaveTitle('Other — inertia-go · Solid template')
  await expect(page.locator('meta[name="description"]')).toHaveCount(1);await expect(page.locator('meta[name="description"]')).toHaveAttribute('content','Other')
})
test('F11 merge append and matchOn avoids duplicate',async({page})=>{
  await open(page,'/merge');await visit(page,'/merge?page=2',{only:['items'],preserveState:true});expect((await props(page)).items.map((x:any)=>x.id)).toEqual([1,2])
  await visit(page,'/merge?page=2',{only:['items'],preserveState:true});expect((await props(page)).items.map((x:any)=>x.id)).toEqual([1,2])
})
test('F11 merge reset and full visit replace',async({page})=>{
  await open(page,'/merge');await visit(page,'/merge?page=2',{only:['items']});await visit(page,'/merge?page=3',{only:['items'],reset:['items']});expect((await props(page)).items.map((x:any)=>x.id)).toEqual([3])
  await visit(page,'/merge?page=1');expect((await props(page)).items.map((x:any)=>x.id)).toEqual([1])
})
test('F11 deep merge nested arrays',async({page})=>{
  await open(page,'/deep');await visit(page,'/deep?page=2',{only:['tree']});expect((await props(page)).tree.items.map((x:any)=>x.id)).toEqual([1,2])
})
test('F12 local manual pagination uses native scroll metadata',async({page})=>{
  await open(page,'/scroll');expect(await page.evaluate(()=>window.__lab.scroll?.hasNext())).toBe(true)
  await page.evaluate(()=>window.__lab.scroll.fetchNext());await expect.poll(async()=> (await props(page)).items.data.map((x:any)=>x.id)).toEqual([1,2])
})
test('F13 WhenVisible waits then loads optional exactly once',async({page})=>{
  await open(page,'/visible');expect((await stats(page)).evaluations.optional??0).toBe(0)
  await page.locator('#visible-fallback').scrollIntoViewIfNeeded();await expect(page.locator('#visible-value')).toHaveText('optional-value')
  expect((await stats(page)).evaluations.optional).toBe(1)
})
test('F14 polling start stop and cleanup on unmount',async({page})=>{
  await open(page,'/poll');await page.locator('#poll-start').click();await expect.poll(async()=> (await stats(page)).evaluations.tick).toBeGreaterThan(2)
  await page.locator('#poll-stop').click();await pause(page,180);const stopped=(await stats(page)).evaluations.tick;await pause(page,250);expect((await stats(page)).evaluations.tick).toBe(stopped)
  await page.locator('#poll-start').click();await visit(page,'/other');await pause(page,180);const left=(await stats(page)).evaluations.tick;await pause(page,250);expect((await stats(page)).evaluations.tick).toBe(left)
})
test('F15 useForm dirty reset and successful lifecycle',async({page})=>{
  await open(page,'/form');await page.locator('#form-name').fill('Ada');expect(JSON.parse(await page.locator('#form-state').innerText()).dirty).toBe(true)
  await page.locator('#form-reset').click();await expect(page.locator('#form-name')).toHaveValue('')
  await page.locator('#form-name').fill('Ada');await page.locator('#form-submit').click()
  await expect.poll(async()=> JSON.parse(await page.locator('#form-state').innerText()).success).toBe(true)
  expect(JSON.parse(await page.locator('#form-state').innerText()).processing).toBe(false)
})
test('F15 Form component serializes and succeeds',async({page})=>{
  await open(page,'/form');await page.locator('#native-name').fill('Ada');await page.locator('#native-submit').click()
  await expect.poll(async()=>JSON.parse(await page.locator('#native-state').innerText()).success).toBe(true)
})
test('F16 validation redirect populates useForm errors',async({page})=>{
  await open(page,'/form');await page.locator('#form-submit').click()
  await expect.poll(async()=> JSON.parse(await page.locator('#form-state').innerText()).errors.name).toBe('Name is required')
  expect(await page.evaluate(()=>window.__lab.events)).toContain('error')
})
test('F16 named error bag',async({page})=>{
  await open(page,'/form');await page.evaluate(()=>window.__lab.form.get('/form?bag=1',{errorBag:'profile',preserveState:true}))
  await expect.poll(async()=>JSON.parse(await page.locator('#form-state').innerText()).errors.name).toBe('Name is required')
})
test('F17 useForm Precognition invalid and valid fields',async({page})=>{
  await open(page,'/form');await page.locator('#precognition-invalid').click()
  await expect.poll(async()=>JSON.parse(await page.locator('#precognition-state').innerText()).invalid).toBe(true)
  await page.locator('#precognition-valid').click();await expect.poll(async()=>JSON.parse(await page.locator('#precognition-state').innerText()).valid).toBe(true)
  expect((await requests(page,'/submit')).length).toBe(0)
})
test('F18 useHttp JSON request preserves page and history',async({page})=>{
  await open(page,'/form');const history=await page.evaluate(()=>history.length);await page.locator('#http-request').click()
  await expect.poll(async()=>JSON.parse(await page.locator('#http-state').innerText()).response).toEqual({ok:true,value:42})
  await expect(page.locator('#component')).toHaveText('FormPage');expect(await page.evaluate(()=>window.history.length)).toBe(history)
})
test('F19 history encryption and restoration',async({page})=>{
  await open(page,'/history');await expect.poll(()=>page.evaluate(()=>history.state?.page instanceof ArrayBuffer)).toBe(true)
  await visit(page,'/other');await page.goBack();await expect(page.locator('#component')).toHaveText('HistoryPage')
})
test('F19 server clearHistory flag',async({page})=>{
  await open(page,'/home');await visit(page,'/clear-history');await expect(page.locator('#component')).toHaveText('HistoryPage')
  expect(await page.evaluate(()=>window.__lab.page().clearHistory)).toBe(true)
})
test('F20 client push replace and prop helpers without server requests',async({page})=>{
  await open(page);const before=(await stats(page)).requests.length
  await page.evaluate(()=>window.__lab.router.push({url:'/local',component:'Other',props:{label:'local',items:[1]}}));await expect(page.locator('#component')).toHaveText('Other')
  await page.evaluate(()=>window.__lab.router.replaceProp('label','changed'));await expect.poll(async()=> (await props(page)).label).toBe('changed')
  await page.evaluate(()=>window.__lab.router.appendToProp('items',2));await expect.poll(async()=> (await props(page)).items).toEqual([1,2])
  await page.evaluate(()=>window.__lab.router.prependToProp('items',0));await expect.poll(async()=> (await props(page)).items).toEqual([0,1,2])
  expect((await stats(page)).requests.length).toBe(before)
})
test('F21 successful visit event lifecycle',async({page})=>{
  await open(page);await page.evaluate(()=>window.__lab.events.length=0);await visit(page,'/other')
  expect(await page.evaluate(()=>window.__lab.events)).toEqual(expect.arrayContaining(['before','start','success','finish','navigate']))
})
test('F21 cancelAll prevents delayed page replacement',async({page})=>{
  await open(page);await page.evaluate(()=>window.__lab.router.visit('/other?delay=600'));await pause(page,80);await page.evaluate(()=>window.__lab.router.cancelAll())
  await pause(page,700);await expect(page.locator('#component')).toHaveText('Home')
})
for(const method of ['post','put','patch','delete'])test(`F05 ${method.toUpperCase()} follows redirect as GET`,async({page})=>{
  await open(page);await visit(page,'/submit',{method,data:{name:'Ada'}});await expect(page.locator('#component')).toHaveText('FormPage')
  expect((await stats(page)).requests.some((r:any)=>r.path.startsWith('/form')&&r.method==='GET')).toBe(true)
})
test('F05 server redirect and location full refresh',async({page})=>{
  await open(page);await visit(page,'/redirect');await expect(page.locator('#component')).toHaveText('Other')
  const boot=await page.evaluate(()=>window.__lab.boot)
  await Promise.all([page.waitForEvent('load'),page.evaluate(()=>{window.__lab.router.visit('/location')})])
  await page.waitForFunction(()=>window.__lab?.ready);expect(await page.evaluate(()=>window.__lab.boot)).not.toBe(boot);await expect(page.locator('#component')).toHaveText('Other')
})
test('F06 useRemember restores input through history',async({page})=>{
  await open(page);await page.locator('#remembered').fill('remember-me');await visit(page,'/other');await page.goBack();await expect(page.locator('#remembered')).toHaveValue('remember-me')
})
for(const status of [403,404,500])test(`F22 HTTP ${status} reports exception and preserves page`,async({page})=>{
  await open(page);await page.evaluate(()=>{window.__lab.httpErrors=[];window.__lab.router.on('httpException',(e:any)=>{window.__lab.httpErrors.push(e.detail.response.status);return false})})
  await visit(page,'/failure?status='+status);expect(await page.evaluate(()=>window.__lab.httpErrors)).toContain(status);await expect(page.locator('#component')).toHaveText('Home')
})
test('F22 offline reports networkError and preserves page',async({page,context})=>{
  await open(page);await page.evaluate(()=>{window.__lab.networkErrors=0;window.__lab.router.on('networkError',()=>{window.__lab.networkErrors++;return false})})
  await context.setOffline(true);await visit(page,'/other');expect(await page.evaluate(()=>window.__lab.networkErrors)).toBe(1);await expect(page.locator('#component')).toHaveText('Home');await context.setOffline(false)
})
test('F22 malformed non-Inertia JSON is surfaced',async({page})=>{
  await open(page);await page.evaluate(()=>{window.__lab.httpErrors=[];window.__lab.router.on('httpException',(e:any)=>{window.__lab.httpErrors.push(e.detail.response.status);return false})})
  await visit(page,'/malformed');expect(await page.evaluate(()=>window.__lab.httpErrors)).toContain(200);await expect(page.locator('#component')).toHaveText('Home')
})

test('F01 local prefetch wrapper includes query URL',async({page})=>{
  await open(page);await page.evaluate(()=>window.__lab.router.prefetch('/other?case=hook'))
  await expect.poll(()=>page.evaluate(()=>!!window.__lab.router.getCached('/other?case=hook'))).toBe(true)
  await visit(page,'/other?case=hook');expect(await page.evaluate(()=>window.__lab.prefetch.isPrefetched)).toBe(true)
})
test('F01 concurrent prefetch and navigation cannot replace current page',async({page})=>{
  await open(page);await page.evaluate(()=>window.__lab.router.prefetch('/other?delay=300'))
  await visit(page,'/props');await pause(page,450);await expect(page.locator('#component')).toHaveText('Props')
})
test('F01 prefetch HTTP failure preserves current page',async({page})=>{
  await open(page);await page.evaluate(()=>{window.__lab.prefetchFailures=[];window.__lab.router.on('httpException',(e:any)=>{window.__lab.prefetchFailures.push(e.detail.response.status);return false});window.__lab.router.prefetch('/failure?status=403')})
  await expect.poll(()=>page.evaluate(()=>!!window.__lab.router.getCached('/failure?status=403'))).toBe(true)
  await expect(page.locator('#component')).toHaveText('Home')
  await visit(page,'/failure?status=403');expect(await page.evaluate(()=>window.__lab.prefetchFailures)).toContain(403)
  await expect(page.locator('#component')).toHaveText('Home');expect(await requests(page,'/failure?status=403')).toHaveLength(1)
})
test('F02 concurrent navigation prevents stale delayed response',async({page})=>{
  await open(page);await page.evaluate(()=>window.__lab.router.visit('/other?delay=300'));await pause(page,50)
  await visit(page,'/props');await pause(page,400);await expect(page.locator('#component')).toHaveText('Props')
})
test('F06 preserveState false resets same-component local state',async({page})=>{
  await open(page,'/props');await page.locator('#local-state').fill('changed');await visit(page,'/props',{preserveState:false})
  await expect(page.locator('#local-state')).toHaveValue('initial')
})
test('F15 useForm file upload serialization and progress',async({page})=>{
  await open(page,'/form')
  await page.evaluate(()=>{const f=window.__lab.form;f.setData('name','Ada');f.setData('file',new File(['payload'],'fixture.txt',{type:'text/plain'}));window.__lab.uploadEvents=[];f.post('/submit',{forceFormData:true,onProgress:(p:any)=>window.__lab.uploadEvents.push(p.percentage)})})
  await expect.poll(async()=>JSON.parse(await page.locator('#form-state').innerText()).success).toBe(true)
  expect(await page.evaluate(()=>window.__lab.uploadEvents.length)).toBeGreaterThan(0)
})
test('F16 preserveErrors partial background reload',async({page})=>{
  await open(page,'/form?invalid=1');expect((await props(page)).errors.name).toBe('Name is required')
  await visit(page,'/form',{only:['errors'],preserveState:true,preserveErrors:true})
  expect((await props(page)).errors.name).toBe('Name is required')
})
test('F18 useHttp Promise resolves JSON without page replacement',async({page})=>{
  await open(page,'/form');const length=await page.evaluate(()=>history.length)
  const result=await page.evaluate(()=>window.__lab.http.get('/http'))
  expect(result).toEqual({ok:true,value:42});await expect(page.locator('#component')).toHaveText('FormPage');expect(await page.evaluate(()=>history.length)).toBe(length)
})
test('F20 client replace does not add history entry',async({page})=>{
  await open(page);const length=await page.evaluate(()=>history.length)
  await page.evaluate(()=>window.__lab.router.replace({url:'/replaced',component:'Other',props:{label:'replaced'}}))
  await expect(page.locator('#component')).toHaveText('Other');expect(await page.evaluate(()=>history.length)).toBe(length)
})
test('F21 before handler can cancel visit',async({page})=>{
  await open(page);await page.evaluate(()=>window.__lab.router.on('before',()=>false))
  await page.locator('#nav-other').click();await pause(page,180);await expect(page.locator('#component')).toHaveText('Home');expect(await requests(page,'/other')).toHaveLength(0)
})
