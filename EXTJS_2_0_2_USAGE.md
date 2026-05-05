# Ext JS 2.0.2 Usage Reference

This is a compact reference for using the Ext JS 2.0.2 library bundled with
HorizonJS. It is meant for humans and coding agents. Treat this codebase as Ext
JS 2.0.2, not modern Sencha Ext JS.

## Agent Rules

- Use Ext 2 APIs only. Do not use `Ext.define`, `Ext.create`, `requires`,
  controllers, models, ViewModels, modern stores, or Ext 4+ class syntax.
- Prefer the bundled API docs and examples over memory or internet examples:
  `extjs-2.0.2/docs/index.html`, `build/docs/index.html`,
  `extjs-2.0.2/examples/`, and `build/examples/`.
- Write classic constructor/config JavaScript: `new Ext.Panel({...})`,
  `new Ext.data.Store({...})`, `Ext.extend(...)`, `Ext.reg(...)`, `var`, and
  function expressions.
- Load scripts with plain script tags in dependency order. Ext 2 has no module
  loader or bundler contract.
- Keep `extjs-2.0.2/` pristine when changing HorizonJS itself. Library patches
  should normally be placed under `overlay/` so the build process overlays them.
- Remember that Ajax and Store loads are asynchronous. Put follow-up work in
  callbacks or event listeners.

## Where Things Are

- Original library source: `extjs-2.0.2/source/`
- Original API docs: `extjs-2.0.2/docs/index.html`
- Original examples: `extjs-2.0.2/examples/`
- Built distributable: `build/build/`
- Built API docs: `build/docs/index.html`
- Built examples: `build/examples/`
- Main built files: `build/build/ext-all.js`, `build/build/ext-all-debug.js`,
  `build/build/ext-core.js`, `build/build/ext-core-debug.js`
- CSS and images: `build/build/resources/css/ext-all.css` and
  `build/build/resources/images/`

Use `ext-all-debug.js` while developing and `ext-all.js` for production. Use
`ext-core.js` only for DOM, events, Ajax, templates, and utilities; most widgets
such as panels, grids, forms, trees, windows, layouts, and toolbars require
`ext-all.js`.

## Include Order

Recommended standalone order:

```html
<link rel="stylesheet" type="text/css" href="/ext/resources/css/ext-all.css">

<script type="text/javascript" src="/ext/adapter/ext/ext-base.js"></script>
<script type="text/javascript" src="/ext/ext-all-debug.js"></script>
<script type="text/javascript">
Ext.BLANK_IMAGE_URL = '/ext/resources/images/default/s.gif';

Ext.onReady(function () {
    // Build Ext UI here.
});
</script>
```

When copying Ext files into an app, copy the whole distribution directory or keep
the same relative relationship between `resources/css/` and `resources/images/`.
The CSS references image assets.

Alternative adapters are included for old YUI, jQuery, and Prototype stacks. Use
them only when an existing app requires them:

```text
Standalone:  adapter/ext/ext-base.js, then ext-all.js
YUI:         yui-utilities.js, ext-yui-adapter.js, then ext-all.js
jQuery:      jquery.js, ext-jquery-adapter.js, then ext-all.js
Prototype:   prototype.js, scriptaculous.js, ext-prototype-adapter.js, then ext-all.js
```

## Basic Application Shape

Ext code starts after the DOM is ready:

```javascript
Ext.onReady(function () {
    Ext.QuickTips.init();
    Ext.form.Field.prototype.msgTarget = 'side';

    new Ext.Panel({
        renderTo: 'app',
        title: 'Hello',
        width: 400,
        height: 160,
        bodyStyle: 'padding:10px',
        html: 'Ext JS 2.0.2 is running.'
    });
});
```

For full-page apps, use `Ext.Viewport`:

```javascript
Ext.onReady(function () {
    new Ext.Viewport({
        layout: 'border',
        items: [{
            region: 'north',
            height: 40,
            html: '<h1>Application</h1>'
        }, {
            region: 'west',
            title: 'Navigation',
            width: 220,
            split: true,
            collapsible: true,
            layout: 'accordion',
            items: [{
                title: 'Section',
                html: 'Links go here'
            }]
        }, {
            region: 'center',
            xtype: 'tabpanel',
            activeTab: 0,
            items: [{
                title: 'Home',
                html: 'Main content'
            }]
        }]
    });
});
```

## Core Concepts

- `Ext` is the global namespace.
- `Ext.onReady(fn, scope)` waits for the DOM and Ext initialization.
- `Ext.get(idOrNode)` returns a cached `Ext.Element`; keep it when you need a
  stable wrapper.
- `Ext.fly(idOrNode)` returns a shared flyweight `Ext.Element`; use it for
  one-off DOM work and do not store it.
- `Ext.getCmp(id)` returns a registered component by id.
- `Ext.apply(target, source)` copies properties. `Ext.applyIf(target, source)`
  copies only missing properties.
- `Ext.ns('App', 'App.view')` creates namespaces.
- `Ext.encode(obj)` and `Ext.decode(text)` are JSON helpers.
- `Ext.urlEncode(obj)` and `Ext.urlDecode(query)` handle query strings.

Useful DOM helpers:

```javascript
var el = Ext.get('message');
el.update('Saved');
el.addClass('is-saved');
el.on('click', function (event, target) {
    event.stopEvent();
});

Ext.DomHelper.append('list', {
    tag: 'li',
    cls: 'item',
    html: 'Created by DomHelper'
});

Ext.select('.row').addClass('highlight');
```

## Components, Containers, and Rendering

Most UI objects are `Ext.Component` subclasses. Components can be rendered in
three common ways:

```javascript
new Ext.Panel({ renderTo: 'target-id', title: 'Rendered immediately' });

var panel = new Ext.Panel({ title: 'Rendered later' });
panel.render('target-id');

new Ext.Panel({
    applyTo: 'existing-markup-id',
    title: 'Enhances existing markup'
});
```

Containers manage child components through `items`. Child configs can use
`xtype` for lazy creation:

```javascript
new Ext.Panel({
    renderTo: 'app',
    layout: 'form',
    defaults: { width: 250 },
    defaultType: 'textfield',
    items: [{
        fieldLabel: 'Name',
        name: 'name',
        allowBlank: false
    }, {
        xtype: 'datefield',
        fieldLabel: 'Start',
        name: 'start'
    }]
});
```

Common component configs:

- Identity and content: `id`, `itemId`, `title`, `html`, `contentEl`, `cls`,
  `bodyStyle`, `iconCls`
- Sizing and scrolling: `width`, `height`, `autoHeight`, `autoWidth`,
  `autoScroll`
- Rendering: `renderTo`, `el`, `applyTo`, `autoEl`
- Container behavior: `layout`, `layoutConfig`, `items`, `defaults`,
  `defaultType`
- Tool areas: `tbar`, `bbar`, `buttons`, `tools`
- Behavior: `listeners`, `scope`, `plugins`, `disabled`, `hidden`,
  `collapsible`, `closable`

Common layouts:

- `fit`: one child fills the container.
- `border`: full app regions: `north`, `south`, `east`, `west`, `center`.
- `column`: columns with `columnWidth` or fixed `width`.
- `anchor`: child sizing using anchors such as `95%` or `-20`.
- `form`: label and field layout for forms.
- `table`: table cells with `layoutConfig: {columns: n}`.
- `accordion`: collapsible stacked panels.
- `card`: multiple cards with one active item, often driven by tabs/wizards.
- `absolute`: positioned children with `x` and `y`.

## Common XTypes

Use these in child component configs:

```text
box              Ext.BoxComponent
button           Ext.Button
colorpalette     Ext.ColorPalette
component        Ext.Component
container        Ext.Container
cycle            Ext.CycleButton
dataview         Ext.DataView
datepicker       Ext.DatePicker
editor           Ext.Editor
editorgrid       Ext.grid.EditorGridPanel
grid             Ext.grid.GridPanel
paging           Ext.PagingToolbar
panel            Ext.Panel
progress         Ext.ProgressBar
splitbutton      Ext.SplitButton
tabpanel         Ext.TabPanel
treepanel        Ext.tree.TreePanel
viewport         Ext.Viewport
window           Ext.Window

toolbar          Ext.Toolbar
tbbutton         Ext.Toolbar.Button
tbfill           Ext.Toolbar.Fill
tbitem           Ext.Toolbar.Item
tbseparator      Ext.Toolbar.Separator
tbspacer         Ext.Toolbar.Spacer
tbsplit          Ext.Toolbar.SplitButton
tbtext           Ext.Toolbar.TextItem

form             Ext.FormPanel
checkbox         Ext.form.Checkbox
combo            Ext.form.ComboBox
datefield        Ext.form.DateField
field            Ext.form.Field
fieldset         Ext.form.FieldSet
hidden           Ext.form.Hidden
htmleditor       Ext.form.HtmlEditor
numberfield      Ext.form.NumberField
radio            Ext.form.Radio
textarea         Ext.form.TextArea
textfield        Ext.form.TextField
timefield        Ext.form.TimeField
trigger          Ext.form.TriggerField
```

Register custom component classes with `Ext.reg(xtype, constructor)`.

## Events

Components and many utilities extend `Ext.util.Observable`.

```javascript
var win = new Ext.Window({
    title: 'Editor',
    width: 500,
    height: 300,
    listeners: {
        show: function (windowRef) {
            windowRef.body.update('Shown');
        },
        scope: this
    }
});

win.on('hide', function () {
    // Runs when hidden.
});
```

For custom events in subclasses:

```javascript
App.TaskPanel = Ext.extend(Ext.Panel, {
    initComponent: function () {
        App.TaskPanel.superclass.initComponent.call(this);
        this.addEvents('taskcomplete');
    },

    completeTask: function (record) {
        this.fireEvent('taskcomplete', this, record);
    }
});
```

## Subclassing and Plugins

Use `Ext.extend` and call the superclass method explicitly.

```javascript
Ext.ns('App');

App.UserGrid = Ext.extend(Ext.grid.GridPanel, {
    initComponent: function () {
        var store = new Ext.data.JsonStore({
            url: '/api/users',
            root: 'users',
            id: 'id',
            fields: ['id', 'name', 'email'],
            autoLoad: true
        });

        Ext.apply(this, {
            store: store,
            columns: [
                {header: 'Name', dataIndex: 'name', sortable: true},
                {id: 'email', header: 'Email', dataIndex: 'email', sortable: true}
            ],
            autoExpandColumn: 'email',
            loadMask: true
        });

        App.UserGrid.superclass.initComponent.call(this);
    }
});

Ext.reg('usergrid', App.UserGrid);
```

Plugins are objects with an `init(component)` method:

```javascript
App.FocusPlugin = {
    init: function (cmp) {
        cmp.on('afterrender', function () {
            if (cmp.focus) {
                cmp.focus();
            }
        });
    }
};
```

Use as `plugins: App.FocusPlugin` or `plugins: [App.FocusPlugin]`.

## Ajax

Use `Ext.Ajax.request`. Methods are uppercase. If `method` is omitted, Ext uses
`POST` when params exist and `GET` otherwise.

```javascript
Ext.Ajax.request({
    url: '/api/save',
    method: 'POST',
    params: {
        id: 12,
        name: 'Ada'
    },
    success: function (response, options) {
        var data = Ext.decode(response.responseText);
        Ext.Msg.alert('Saved', data.message || 'Done');
    },
    failure: function (response) {
        Ext.Msg.alert('Error', 'Request failed: ' + response.status);
    },
    scope: this
});
```

`Ext.data.HttpProxy` and `Ext.Ajax` are same-origin XHR. For legacy
cross-domain JSON-style loading, use `Ext.data.ScriptTagProxy` and a server that
supports the expected callback parameter.

## Data Stores and Readers

Stores hold `Ext.data.Record` instances. A proxy fetches raw data, a reader
turns it into records, and UI components such as grids consume the store.

Field configs can include `name`, `mapping`, `type`, `dateFormat`, `sortType`,
`sortDir`, `defaultValue`, and `convert`. Supported field types include `auto`,
`string`, `int`, `float`, `bool`/`boolean`, and `date`.

Inline array data:

```javascript
var store = new Ext.data.SimpleStore({
    fields: [
        {name: 'company'},
        {name: 'price', type: 'float'},
        {name: 'lastChange', type: 'date', dateFormat: 'n/j h:ia'}
    ]
});

store.loadData([
    ['Acme', 12.34, '5/1 9:00am'],
    ['ExampleCo', 56.78, '5/2 10:30am']
]);
```

Remote JSON:

```javascript
var store = new Ext.data.Store({
    proxy: new Ext.data.HttpProxy({
        url: '/api/topics'
    }),
    reader: new Ext.data.JsonReader({
        root: 'topics',
        totalProperty: 'totalCount',
        id: 'id',
        fields: [
            'title',
            'author',
            {name: 'replyCount', mapping: 'reply_count', type: 'int'},
            {name: 'lastPost', mapping: 'last_post', type: 'date', dateFormat: 'timestamp'}
        ]
    }),
    remoteSort: true
});

store.setDefaultSort('lastPost', 'DESC');
store.load({
    params: {start: 0, limit: 25},
    callback: function (records, options, success) {
        if (!success) {
            Ext.Msg.alert('Error', 'Could not load data');
        }
    }
});
```

Store events: `beforeload`, `load`, `loadexception`, `datachanged`, `add`,
`remove`, `update`, and `metachange`.

## Grids

`Ext.grid.GridPanel` needs a store and columns. Use `Ext.grid.EditorGridPanel`
for editable cells.

```javascript
var grid = new Ext.grid.GridPanel({
    renderTo: 'grid',
    title: 'Users',
    width: 700,
    height: 400,
    store: store,
    columns: [
        {id: 'name', header: 'Name', dataIndex: 'name', sortable: true},
        {header: 'Email', dataIndex: 'email', sortable: true},
        {header: 'Joined', dataIndex: 'joined', renderer: Ext.util.Format.dateRenderer('m/d/Y')}
    ],
    autoExpandColumn: 'name',
    stripeRows: true,
    loadMask: true,
    bbar: new Ext.PagingToolbar({
        pageSize: 25,
        store: store,
        displayInfo: true
    })
});
```

Useful grid examples:

- `extjs-2.0.2/examples/grid/array-grid.js`
- `extjs-2.0.2/examples/grid/paging.js`
- `extjs-2.0.2/examples/grid/edit-grid.js`
- `extjs-2.0.2/examples/grid/grouping.js`

## Forms

Use `Ext.FormPanel` for field layout and `formPanel.getForm()` for the
underlying `Ext.form.BasicForm`.

```javascript
var form = new Ext.FormPanel({
    renderTo: 'form',
    title: 'Profile',
    url: '/api/profile',
    frame: true,
    labelWidth: 90,
    width: 380,
    bodyStyle: 'padding:8px',
    defaults: {width: 240},
    defaultType: 'textfield',
    items: [{
        fieldLabel: 'Name',
        name: 'name',
        allowBlank: false
    }, {
        fieldLabel: 'Email',
        name: 'email',
        vtype: 'email'
    }, {
        xtype: 'datefield',
        fieldLabel: 'Birthday',
        name: 'birthday',
        format: 'm/d/Y'
    }],
    buttons: [{
        text: 'Save',
        handler: function () {
            form.getForm().submit({
                waitMsg: 'Saving...',
                success: function (basicForm, action) {
                    Ext.Msg.alert('Saved', 'Profile updated');
                },
                failure: function (basicForm, action) {
                    Ext.Msg.alert('Error', 'Save failed');
                }
            });
        }
    }]
});
```

Default JSON response shapes:

```javascript
// Submit success or failure.
{success: true}
{success: false, errors: {email: 'Invalid email'}}

// Load form data.
{success: true, data: {name: 'Ada', email: 'ada@example.com'}}
```

Useful form examples:

- `extjs-2.0.2/examples/form/dynamic.js`
- `extjs-2.0.2/examples/form/combos.js`
- `extjs-2.0.2/examples/form/form-grid.js`
- `extjs-2.0.2/examples/form/xml-form.js`

## Trees

Tree nodes are `Ext.tree.TreeNode` or `Ext.tree.AsyncTreeNode`. Use
`Ext.tree.TreeLoader` for remote children.

```javascript
var tree = new Ext.tree.TreePanel({
    renderTo: 'tree',
    title: 'Folders',
    width: 260,
    height: 360,
    rootVisible: false,
    autoScroll: true,
    loader: new Ext.tree.TreeLoader({
        dataUrl: '/api/tree'
    }),
    root: new Ext.tree.AsyncTreeNode({
        text: 'Root',
        id: 'root',
        expanded: true
    })
});
```

Remote tree data is an array of node configs, commonly with `id`, `text`,
`leaf`, `children`, `expanded`, `cls`, and `iconCls`.

Useful tree examples:

- `extjs-2.0.2/examples/tree/reorder.js`
- `extjs-2.0.2/examples/tree/two-trees.js`
- `extjs-2.0.2/examples/tree/column-tree.js`

## Tabs, Windows, Menus, and Messages

```javascript
var tabs = new Ext.TabPanel({
    renderTo: 'tabs',
    activeTab: 0,
    width: 600,
    height: 250,
    defaults: {autoScroll: true},
    items: [{
        title: 'Local',
        html: 'Local content'
    }, {
        title: 'Ajax',
        autoLoad: {url: '/partial.html', params: 'id=1'}
    }]
});

var win = new Ext.Window({
    title: 'Details',
    width: 420,
    height: 260,
    modal: true,
    layout: 'fit',
    items: {xtype: 'panel', html: 'Window body'}
});

win.show();
Ext.Msg.alert('Status', 'Ready');
```

Menus are usually configured on buttons:

```javascript
new Ext.Button({
    renderTo: 'actions',
    text: 'Actions',
    menu: [{
        text: 'Refresh',
        handler: function () {
            store.reload();
        }
    }]
});
```

## Templates

Use `Ext.Template` for simple substitution and `Ext.XTemplate` for loops and
conditions.

```javascript
var tpl = new Ext.Template(
    '<p>Name: {name}</p>',
    '<p>Email: {email}</p>'
);
tpl.overwrite('target', {name: 'Ada', email: 'ada@example.com'});

var listTpl = new Ext.XTemplate(
    '<ul>',
    '<tpl for="items">',
    '<li>{#}. {name}</li>',
    '</tpl>',
    '</ul>'
);
listTpl.overwrite('target', {items: [{name: 'One'}, {name: 'Two'}]});
```

See `extjs-2.0.2/examples/core/templates.js`.

## State

To persist component state such as grid column settings, configure a state
provider and give stateful components stable ids.

```javascript
Ext.state.Manager.setProvider(new Ext.state.CookieProvider());

new Ext.grid.GridPanel({
    id: 'users-grid',
    stateful: true,
    store: store,
    columns: columns
});
```

## Locales

Locale files live under `extjs-2.0.2/source/locale/` and examples under
`extjs-2.0.2/examples/locale/`. Load the locale file after `ext-all.js` and
before building UI.

```html
<script type="text/javascript" src="/ext/ext-all.js"></script>
<script type="text/javascript" src="/ext/source/locale/ext-lang-fr.js"></script>
```

If using the built distribution, verify the locale file is copied to the served
location first.

## Debugging and Verification

- Use `ext-all-debug.js` when diagnosing stack traces.
- Check generated docs for exact method signatures.
- Run examples from a local web server when they use Ajax, XML, JSON, or PHP
  placeholders. Some static examples can open directly, but Ajax examples need
  HTTP.
- Component layout bugs often come from missing `width`, `height`, `layout`, or
  parent dimensions. Grids and fit/border layouts especially need real sizes.
- If images are broken, verify `Ext.BLANK_IMAGE_URL` and that CSS image paths
  still point to `resources/images/`.
- Destroy windows, panels, and custom components when removing them manually:
  `component.destroy()`.

## Common Pitfalls

- `xtype` works for child component configs inside containers. For standalone
  construction, prefer `new Ext.SomeClass(config)`.
- `contentEl` moves or uses existing markup as component content. Hide source
  markup with `class="x-hide-display"` when appropriate.
- `autoLoad` on panels/tabs loads HTML fragments asynchronously.
- `Store.load()` returns before data arrives. Use `callback` or the `load`
  event.
- `remoteSort: true` sends sorting parameters to the server; the server must
  return sorted data.
- `HttpProxy` is same-origin. Use `ScriptTagProxy` only for legacy JSONP-style
  endpoints.
- Ext 2 extends native prototypes (`Function`, `String`, `Number`, `Array`,
  `Date`). Avoid mixing it blindly into strict modern app shells.
- Ext 2 examples often use `iso-8859-1` meta tags, old doctypes, and PHP sample
  endpoints. Adapt paths and endpoints, but keep the Ext API style.

## What To Search First

When implementing a feature with this library, inspect these local examples:

```text
Layout/app shell:    examples/layout/complex.html
Tabs:                examples/tabs/tabs-example.js
Forms:               examples/form/dynamic.js
Combo boxes:         examples/form/combos.js
Grids:               examples/grid/array-grid.js, examples/grid/paging.js
Editable grids:      examples/grid/edit-grid.js
Trees:               examples/tree/reorder.js
DataView:            examples/view/
Windows/dialogs:     examples/window/
Menus/toolbars:      examples/menu/
Drag and drop:       examples/organizer/organizer.js, examples/portal/Portal.js,
                     examples/tree/two-trees.js
Templates:           examples/core/templates.js
State:               examples/state/
```

For exact API details, open the matching class in `docs/output/`, for example:
`Ext.Panel.html`, `Ext.grid.GridPanel.html`, `Ext.data.Store.html`,
`Ext.form.FormPanel.html`, or `Ext.tree.TreePanel.html`.
