resource "truenas_nvmet_host" "initiator" {
  hostnqn     = "nqn.2014-08.org.nvmexpress:uuid:my-initiator"
  dhchap_hash = "SHA-256"
}
