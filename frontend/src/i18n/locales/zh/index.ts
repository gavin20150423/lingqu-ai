import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import batchImage from './batchImage'
import admin from './admin'
import misc from './misc'
import store from './store'
import conversations from './conversations'

export default {
  ...landing,
  ...common,
  ...dashboard,
  ...batchImage,
  ...store,
  ...conversations,
  admin,
  ...misc,
}
